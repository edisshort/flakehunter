// Package experiment orchestrates multi-run trials.
// Phase 2: sequential execution + JSON persistence.
// Phase 3: optional JUnit collection for pytest commands.
package experiment

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/edisshort/flakehunter/internal/engine"
	"github.com/edisshort/flakehunter/internal/junit"
	"github.com/edisshort/flakehunter/internal/model"
	"github.com/edisshort/flakehunter/internal/pytestutil"
	"github.com/edisshort/flakehunter/internal/store"
)

// Config is the user-facing plan for one experiment.
type Config struct {
	Command    []string
	WorkingDir string
	Runs       int
	Parallel   int // Phase 5; >1 currently forces sequential with a note
	Timeout    time.Duration
}

// Runner executes experiments using an engine + store.
type Runner struct {
	Store  *store.Store
	Stdout io.Writer
	Stderr io.Writer
}

// Run creates an experiment and executes the command sequentially N times.
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
	if cfg.Parallel > 1 {
		exp.Note = "parallel>1 requested; Phase 5 will add a worker pool. Running sequentially for now."
		fmt.Fprintf(r.Stderr, "warning: %s\n", exp.Note)
	}

	if err := r.Store.CreateExperiment(exp); err != nil {
		return exp, fmt.Errorf("create experiment: %w", err)
	}

	fmt.Fprintf(r.Stdout, "Experiment %s\n", exp.ID)
	fmt.Fprintf(r.Stdout, "  runs:     %d (sequential)\n", cfg.Runs)
	fmt.Fprintf(r.Stdout, "  command:  %v\n", cfg.Command)
	fmt.Fprintf(r.Stdout, "  workdir:  %s\n", absOrDot(cfg.WorkingDir))
	fmt.Fprintf(r.Stdout, "  data:     %s\n\n", r.Store.ExperimentDir(exp.ID))

	usePytest := pytestutil.LooksLikePytest(cfg.Command)
	var runErrors int

	for i := 1; i <= cfg.Runs; i++ {
		if err := ctx.Err(); err != nil {
			exp.Status = model.StatusCancelled
			_ = r.Store.WriteExperiment(exp)
			return exp, err
		}

		fmt.Fprintf(r.Stdout, "[%d/%d] starting...\n", i, cfg.Runs)
		execResult, err := r.oneRun(ctx, exp, cfg, i, usePytest)
		if err != nil {
			runErrors++
			fmt.Fprintf(r.Stderr, "[%d/%d] engine error: %v\n", i, cfg.Runs, err)
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
			fmt.Fprintf(r.Stdout, "[%d/%d] done (%s)\n", i, cfg.Runs, summary)
		}
	}

	if runErrors > 0 {
		exp.Status = model.StatusFailed
	} else {
		exp.Status = model.StatusCompleted
	}
	if err := r.Store.WriteExperiment(exp); err != nil {
		return exp, err
	}
	fmt.Fprintf(r.Stdout, "\nExperiment %s %s\n", exp.ID, exp.Status)
	fmt.Fprintf(r.Stdout, "Inspect: %s\n", r.Store.ExperimentDir(exp.ID))
	return exp, nil
}

func (r *Runner) oneRun(ctx context.Context, exp model.Experiment, cfg Config, runIndex int, usePytest bool) (model.Execution, error) {
	execDir, stdoutPath, stderrPath, junitPath, _, err := r.Store.PrepareExecutionDir(exp.ID, runIndex)
	if err != nil {
		return model.Execution{}, err
	}
	_ = execDir

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

	if err := r.Store.WriteExecution(exp.ID, runIndex, execution); err != nil {
		return execution, err
	}
	if eng.StartError != nil {
		return execution, eng.StartError
	}
	return execution, nil
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
