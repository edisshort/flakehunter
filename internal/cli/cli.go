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

	"github.com/edisshort/flakehunter/internal/experiment"
	"github.com/edisshort/flakehunter/internal/store"
)

// Version is reported by the "version" command.
const Version = "0.3.0-phase3"

// Exit codes used by the CLI.
const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitValidation = 3
	ExitRuntime    = 4
)

// RunConfig holds options for "flakehunter run".
type RunConfig struct {
	Runs       int
	Parallel   int
	Timeout    time.Duration // 0 = none
	WorkingDir string
	Command    []string
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
		Command:    cfg.Command,
		WorkingDir: cfg.WorkingDir,
		Runs:       cfg.Runs,
		Parallel:   cfg.Parallel,
		Timeout:    cfg.Timeout,
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
		fmt.Fprintf(stdout, "  %s  status=%s runs=%d cmd=%s\n",
			id, exp.Status, exp.RunsRequested, strings.Join(exp.Command, " "))
	}
	return ExitOK
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `FlakeHunter — detect and investigate flaky automated tests

Usage:
  flakehunter <command> [flags]

Commands:
  run       Run a repeated test experiment (sequential in Phase 2/3)
  list      List saved experiments under .flakehunter/
  version   Print the FlakeHunter version
  help      Show this help message

Examples:
  flakehunter help
  flakehunter version
  flakehunter run --runs 5 -- go test ./...
  flakehunter run --runs 10 --timeout 2m -- python -m pytest
  flakehunter list

pytest note:
  When the command looks like pytest, FlakeHunter adds a unique
  --junitxml=... per run and stores parsed per-test results.

`)
}

func printRunUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  flakehunter run [flags] -- <command> [args...]

Flags:
  --runs <n>         Number of times to execute the command (required, >= 1)
  --parallel <n>     Max concurrent executions (default 1; pool in Phase 5)
  --timeout <dur>    Per-run timeout (e.g. 30s, 2m). Default: none
  --workdir <path>   Working directory for the command (default: current)

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
	if len(cfg.Command) == 0 {
		return fmt.Errorf("missing test command after -- (example: -- python -m pytest)")
	}
	return nil
}
