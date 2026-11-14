// Package experiment orchestrates multi-run trials.
//
// Phase 2–3: sequential execution + JUnit collection.
// Phase 4: analysis after runs complete.
// Phase 5: bounded worker pool for --parallel N.
package experiment

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/edisshort/flakehunter/internal/analysis"
	"github.com/edisshort/flakehunter/internal/engine"
	"github.com/edisshort/flakehunter/internal/junit"
	"github.com/edisshort/flakehunter/internal/model"
	"github.com/edisshort/flakehunter/internal/pytestutil"
	"github.com/edisshort/flakehunter/internal/report"
	"github.com/edisshort/flakehunter/internal/store"
)

// Config is the user-facing plan for one experiment.
type Config struct {
	Command    []string
	WorkingDir string
	Runs       int
	Parallel   int
	Timeout    time.Duration
	// MinOutcomes overrides analysis threshold (0 = default).
	MinOutcomes int
}

// Runner executes experiments using an engine + store.
type Runner struct {
	Store  *store.Store
	Stdout io.Writer
	Stderr io.Writer
}

type runOutcome struct {
	index int
	exec  model.Execution
	err   error
}

// Run creates an experiment and executes the command N times.
// Parallelism is capped by cfg.Parallel (minimum 1).
func (r *Runner) Run(ctx context.Context, cfg Config) (model.Experiment, error) {
	if r.Stdout == nil {
		r.Stdout = io.Discard
	}
	if r.Stderr == nil {
		r.Stderr = io.Discard
	}
	if r.Store == nil {
		r.Store = store.New(cfg.WorkingDir)
	}
	if cfg.Parallel < 1 {
		cfg.Parallel = 1
	}
	if cfg.Parallel > cfg.Runs {
		cfg.Parallel = cfg.Runs
	}

	now := time.Now()
	exp := model.Experiment{
		ID:            store.NewExperimentID(now),
		CreatedAt:     now.UTC(),
		Command:       append([]string{}, cfg.Command...),
		WorkingDir:    cfg.WorkingDir,
		RunsRequested: cfg.Runs,
		Parallel:      cfg.Parallel,
		TimeoutMS:     cfg.Timeout.Milliseconds(),
		Status:        model.StatusRunning,
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
	}

	if err := r.Store.CreateExperiment(exp); err != nil {
		return exp, fmt.Errorf("create experiment: %w", err)
	}

	mode := "sequential"
	if cfg.Parallel > 1 {
		mode = fmt.Sprintf("parallel=%d", cfg.Parallel)
	}
	fmt.Fprintf(r.Stdout, "Experiment %s\n", exp.ID)
	fmt.Fprintf(r.Stdout, "  runs:     %d (%s)\n", cfg.Runs, mode)
	fmt.Fprintf(r.Stdout, "  command:  %v\n", cfg.Command)
	fmt.Fprintf(r.Stdout, "  workdir:  %s\n", absOrDot(cfg.WorkingDir))
	fmt.Fprintf(r.Stdout, "  data:     %s\n\n", r.Store.ExperimentDir(exp.ID))

	usePytest := pytestutil.LooksLikePytest(cfg.Command)
	runErrors := r.executeAll(ctx, exp, cfg, usePytest)

	if ctx.Err() != nil {
		exp.Status = model.StatusCancelled
	} else if runErrors > 0 {
		exp.Status = model.StatusFailed
	} else {
		exp.Status = model.StatusCompleted
	}
	if err := r.Store.WriteExperiment(exp); err != nil {
		return exp, err
	}

	fmt.Fprintf(r.Stdout, "\nExperiment %s %s\n", exp.ID, exp.Status)
	fmt.Fprintf(r.Stdout, "Inspect: %s\n", r.Store.ExperimentDir(exp.ID))

	if err := r.analyzeAndPrint(exp, cfg); err != nil {
		fmt.Fprintf(r.Stderr, "warning: analysis failed: %v\n", err)
	}
	return exp, ctx.Err()
}

func (r *Runner) executeAll(ctx context.Context, exp model.Experiment, cfg Config, usePytest bool) int {
	jobs := make(chan int)
	outcomes := make(chan runOutcome, cfg.Runs)

	var logMu sync.Mutex
	var wg sync.WaitGroup

	for w := 1; w <= cfg.Parallel; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for runIndex := range jobs {
				if ctx.Err() != nil {
					outcomes <- runOutcome{index: runIndex, err: ctx.Err()}
					continue
				}
				logMu.Lock()
				fmt.Fprintf(r.Stdout, "[%d/%d] starting (worker %d)...\n", runIndex, cfg.Runs, workerID)
				logMu.Unlock()

				execResult, err := r.oneRun(ctx, exp, cfg, runIndex, workerID, usePytest)

				logMu.Lock()
				if err != nil {
					fmt.Fprintf(r.Stderr, "[%d/%d] engine error: %v\n", runIndex, cfg.Runs, err)
				} else {
					summary := fmt.Sprintf("exit=%d duration=%dms", execResult.ExitCode, execResult.DurationMS)
					if execResult.TimedOut {
						summary += " timed_out=true"
					}
					if execResult.CollectError != "" {
						summary += " collect=" + execResult.CollectError
					} else if len(execResult.Tests) > 0 {
						summary += fmt.Sprintf(" tests=%d", len(execResult.Tests))
					}
					fmt.Fprintf(r.Stdout, "[%d/%d] done (%s)\n", runIndex, cfg.Runs, summary)
				}
				logMu.Unlock()

				outcomes <- runOutcome{index: runIndex, exec: execResult, err: err}
			}
		}(w)
	}

dispatch:
	for i := 1; i <= cfg.Runs; i++ {
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	close(outcomes)

	runErrors := 0
	for o := range outcomes {
		if o.err != nil {
			runErrors++
		}
	}
	return runErrors
}

func (r *Runner) oneRun(ctx context.Context, exp model.Experiment, cfg Config, runIndex, workerID int, usePytest bool) (model.Execution, error) {
	_, stdoutPath, stderrPath, junitPath, _, err := r.Store.PrepareExecutionDir(exp.ID, runIndex)
	if err != nil {
		return model.Execution{}, err
	}

	argv := append([]string{}, cfg.Command...)
	if usePytest {
		argv = pytestutil.WithJUnitXML(argv, junitPath)
	}

	stdoutFile, err := engine.OpenLogFile(stdoutPath)
	if err != nil {
		return model.Execution{}, err
	}
	defer stdoutFile.Close()

	stderrFile, err := engine.OpenLogFile(stderrPath)
	if err != nil {
		return model.Execution{}, err
	}
	defer stderrFile.Close()

	started := time.Now().UTC()
	eng := engine.Run(ctx, engine.Request{
		Argv:    argv,
		Dir:     cfg.WorkingDir,
		Timeout: cfg.Timeout,
		Stdout:  stdoutFile,
		Stderr:  stderrFile,
	})
	ended := time.Now().UTC()

	execution := model.Execution{
		ExperimentID: exp.ID,
		RunIndex:     runIndex,
		StartedAt:    started,
		EndedAt:      ended,
		DurationMS:   eng.Duration.Milliseconds(),
		ExitCode:     eng.ExitCode,
		TimedOut:     eng.TimedOut,
		StdoutPath:   stdoutPath,
		StderrPath:   stderrPath,
		WorkerID:     workerID,
	}
	if usePytest {
		execution.JUnitPath = junitPath
	}
	if eng.StartError != nil {
		execution.Error = eng.StartError.Error()
	}

	if usePytest {
		tests, cerr := junit.ParseFile(junitPath)
		if cerr != nil {
			execution.CollectError = cerr.Error()
		} else {
			execution.Tests = tests
		}
	}

	// Each run writes only to its own directory — safe under parallelism.
	if err := r.Store.WriteExecution(exp.ID, runIndex, execution); err != nil {
		return execution, err
	}
	if eng.StartError != nil {
		return execution, eng.StartError
	}
	return execution, nil
}

func (r *Runner) analyzeAndPrint(exp model.Experiment, cfg Config) error {
	execs, err := r.Store.LoadExecutions(exp.ID, exp.RunsRequested)
	if err != nil {
		return err
	}
	th := analysis.DefaultThresholds()
	if cfg.MinOutcomes > 0 {
		th.MinOutcomes = cfg.MinOutcomes
	}
	rep := analysis.Analyze(exp.ID, execs, th)
	if err := r.Store.WriteAnalysis(exp.ID, rep); err != nil {
		return err
	}
	fmt.Fprintln(r.Stdout)
	report.WriteTerminal(r.Stdout, rep)
	fmt.Fprintf(r.Stdout, "Saved analysis: %s\n", filepath.Join(r.Store.ExperimentDir(exp.ID), "analysis.json"))
	return nil
}

func absOrDot(dir string) string {
	if dir == "" {
		dir = "."
	}
	abs, err := os.Getwd()
	if dir == "." && err == nil {
		return abs
	}
	return dir
}
