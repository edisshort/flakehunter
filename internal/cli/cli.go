// Package cli implements FlakeHunter's command-line interface.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/edisshort/flakehunter/internal/analysis"
	"github.com/edisshort/flakehunter/internal/experiment"
	"github.com/edisshort/flakehunter/internal/report"
	"github.com/edisshort/flakehunter/internal/store"
)

// Version is reported by the "version" command.
const Version = "0.5.0-phase5"

// Exit codes used by the CLI.
const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitValidation = 3
	ExitRuntime    = 4
)

// RunConfig holds options for "flakehunter run".
type RunConfig struct {
	Runs        int
	Parallel    int
	Timeout     time.Duration // 0 = none
	WorkingDir  string
	MinOutcomes int // 0 = analysis default
	Command     []string
}

// Run is the CLI entry used by main.
func Run(args []string) int {
	return run(args, os.Stdout, os.Stderr)
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return ExitOK
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return ExitOK
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "flakehunter %s\n", Version)
		return ExitOK
	case "list":
		return cmdList(stdout, stderr)
	case "report":
		return cmdReport(args[1:], stdout, stderr)
	case "run":
		cfg, err := parseRunArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n\n", err)
			printRunUsage(stderr)
			return ExitUsage
		}
		if err := ValidateRunConfig(cfg); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitValidation
		}
		return cmdRun(cfg, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return ExitUsage
	}
}

func cmdRun(cfg RunConfig, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runner := &experiment.Runner{
		Store:  store.New(cfg.WorkingDir),
		Stdout: stdout,
		Stderr: stderr,
	}
	_, err := runner.Run(ctx, experiment.Config{
		Command:     cfg.Command,
		WorkingDir:  cfg.WorkingDir,
		Runs:        cfg.Runs,
		Parallel:    cfg.Parallel,
		Timeout:     cfg.Timeout,
		MinOutcomes: cfg.MinOutcomes,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitRuntime
	}
	return ExitOK
}

func cmdList(stdout, stderr io.Writer) int {
	s := store.New(".")
	ids, err := s.ListExperimentIDs()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitRuntime
	}
	if len(ids) == 0 {
		fmt.Fprintln(stdout, "No experiments yet.")
		fmt.Fprintln(stdout, "Run: flakehunter run --runs 5 -- <command>")
		return ExitOK
	}
	fmt.Fprintln(stdout, "Experiments:")
	for _, id := range ids {
		exp, err := s.ReadExperiment(id)
		if err != nil {
			fmt.Fprintf(stdout, "  %s  (unreadable: %v)\n", id, err)
			continue
		}
		fmt.Fprintf(stdout, "  %s  status=%s runs=%d parallel=%d cmd=%s\n",
			id, exp.Status, exp.RunsRequested, exp.Parallel, strings.Join(exp.Command, " "))
	}
	return ExitOK
}

func cmdReport(args []string, stdout, stderr io.Writer) int {
	s := store.New(".")
	id := ""
	minOutcomes := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--experiment", "-e":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "error: %s requires an id\n", args[i])
				return ExitUsage
			}
			id = args[i+1]
			i++
		case "--min-outcomes":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "error: --min-outcomes requires a value\n")
				return ExitUsage
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 {
				fmt.Fprintf(stderr, "error: --min-outcomes must be >= 1\n")
				return ExitUsage
			}
			minOutcomes = n
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "error: unknown flag %q\n", args[i])
				return ExitUsage
			}
			if id != "" {
				fmt.Fprintf(stderr, "error: unexpected argument %q\n", args[i])
				return ExitUsage
			}
			id = args[i]
		}
	}

	var err error
	if id == "" {
		id, err = s.LatestExperimentID()
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return ExitRuntime
		}
	}

	exp, err := s.ReadExperiment(id)
	if err != nil {
		fmt.Fprintf(stderr, "error: read experiment: %v\n", err)
		return ExitRuntime
	}
	execs, err := s.LoadExecutions(id, exp.RunsRequested)
	if err != nil {
		fmt.Fprintf(stderr, "error: load executions: %v\n", err)
		return ExitRuntime
	}
	th := analysis.DefaultThresholds()
	if minOutcomes > 0 {
		th.MinOutcomes = minOutcomes
	}
	rep := analysis.Analyze(id, execs, th)
	if err := s.WriteAnalysis(id, rep); err != nil {
		fmt.Fprintf(stderr, "error: save analysis: %v\n", err)
		return ExitRuntime
	}
	report.WriteTerminal(stdout, rep)
	fmt.Fprintf(stdout, "Saved analysis: %s\n", s.ExperimentDir(id)+string(os.PathSeparator)+"analysis.json")
	return ExitOK
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `FlakeHunter — detect and investigate flaky automated tests

Usage:
  flakehunter <command> [flags]

Commands:
  run       Run a repeated test experiment (sequential or parallel)
  report    Analyze an experiment and print a flakiness report
  list      List saved experiments under .flakehunter/
  version   Print the FlakeHunter version
  help      Show this help message

Examples:
  flakehunter run --runs 30 -- python -m pytest
  flakehunter run --runs 30 --parallel 4 -- python -m pytest
  flakehunter report
  flakehunter report --experiment 20261114-074011-8400
  flakehunter list

Notes:
  - pytest commands get a unique --junitxml per run.
  - Parallel runs use a bounded worker pool (Phase 5).
  - Classifications are evidence-based, not root-cause claims.

`)
}

func printRunUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  flakehunter run [flags] -- <command> [args...]

Flags:
  --runs <n>           Number of times to execute the command (required, >= 1)
  --parallel <n>       Max concurrent executions (default 1)
  --timeout <dur>      Per-run timeout (e.g. 30s, 2m). Default: none
  --workdir <path>     Working directory for the command (default: current)
  --min-outcomes <n>   Min decisive results before classifying (default 5)

The "--" separator marks the start of the external test command.

`)
}

func parseRunArgs(args []string) (RunConfig, error) {
	cfg := RunConfig{
		Runs:     0,
		Parallel: 1,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "--" {
			cfg.Command = append([]string{}, args[i+1:]...)
			break
		}

		switch arg {
		case "--runs":
			n, err := parseIntFlag("--runs", args, i)
			if err != nil {
				return cfg, err
			}
			cfg.Runs = n
			i += 2
		case "--parallel":
			n, err := parseIntFlag("--parallel", args, i)
			if err != nil {
				return cfg, err
			}
			cfg.Parallel = n
			i += 2
		case "--timeout":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--timeout requires a value (e.g. 30s)")
			}
			d, err := time.ParseDuration(args[i+1])
			if err != nil {
				return cfg, fmt.Errorf("--timeout: %w", err)
			}
			cfg.Timeout = d
			i += 2
		case "--workdir":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--workdir requires a path")
			}
			cfg.WorkingDir = args[i+1]
			i += 2
		case "--min-outcomes":
			n, err := parseIntFlag("--min-outcomes", args, i)
			if err != nil {
				return cfg, err
			}
			if n < 1 {
				return cfg, fmt.Errorf("--min-outcomes must be >= 1")
			}
			cfg.MinOutcomes = n
			i += 2
		default:
			return cfg, fmt.Errorf("unknown flag or argument %q (use -- before the test command)", arg)
		}
	}

	return cfg, nil
}

func parseIntFlag(name string, args []string, i int) (int, error) {
	if i+1 >= len(args) {
		return 0, fmt.Errorf("%s requires a value", name)
	}
	n, err := strconv.Atoi(args[i+1])
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return n, nil
}

// ValidateRunConfig checks that a run configuration is safe to accept.
func ValidateRunConfig(cfg RunConfig) error {
	if cfg.Runs < 1 {
		return fmt.Errorf("--runs must be >= 1 (got %d)", cfg.Runs)
	}
	if cfg.Parallel < 1 {
		return fmt.Errorf("--parallel must be >= 1 (got %d)", cfg.Parallel)
	}
	if cfg.Parallel > cfg.Runs {
		return fmt.Errorf("--parallel (%d) cannot exceed --runs (%d)", cfg.Parallel, cfg.Runs)
	}
	if cfg.Timeout < 0 {
		return fmt.Errorf("--timeout must be >= 0")
	}
	if cfg.MinOutcomes < 0 {
		return fmt.Errorf("--min-outcomes must be >= 0")
	}
	if len(cfg.Command) == 0 {
		return fmt.Errorf("missing test command after -- (example: -- python -m pytest)")
	}
	return nil
}
