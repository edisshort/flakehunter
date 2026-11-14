// Package cli implements FlakeHunter's command-line interface for Phase 1.
//
// Later phases will grow this package (or split it) as we add real
// experiment execution. For now we only parse commands, validate input,
// and print help/version messages.
package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Version is reported by the "version" command.
// We bump this as the MVP grows.
const Version = "0.1.0-phase1"

// Exit codes used by the CLI.
const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitValidation = 3
)

// RunConfig holds options for "flakehunter run".
// Phase 1 only validates these; Phase 2 will execute them.
type RunConfig struct {
	Runs     int
	Parallel int
	Command  []string
}

// Run is the CLI entry used by main.
// args should be os.Args[1:] (everything after the program name).
// stdout/stderr default to the process streams so tests can inject buffers.
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
		// Phase 1: acknowledge the plan. Phase 2 will execute the command.
		fmt.Fprintf(stdout, "FlakeHunter run (Phase 1 stub)\n")
		fmt.Fprintf(stdout, "  runs:     %d\n", cfg.Runs)
		fmt.Fprintf(stdout, "  parallel: %d\n", cfg.Parallel)
		fmt.Fprintf(stdout, "  command:  %s\n", strings.Join(cfg.Command, " "))
		fmt.Fprintf(stdout, "\nExecution engine arrives in Phase 2.\n")
		return ExitOK
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return ExitUsage
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `FlakeHunter — detect and investigate flaky automated tests

Usage:
  flakehunter <command> [flags]

Commands:
  run       Plan a repeated test experiment (execution in Phase 2)
  version   Print the FlakeHunter version
  help      Show this help message

Examples:
  flakehunter help
  flakehunter version
  flakehunter run --runs 30 -- python -m pytest
  flakehunter run --runs 30 --parallel 4 -- python -m pytest

`)
}

func printRunUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  flakehunter run [flags] -- <command> [args...]

Flags:
  --runs <n>       Number of times to execute the command (required, >= 1)
  --parallel <n>   Max concurrent executions (default 1 = sequential)

The "--" separator marks the start of the external test command.

`)
}

// parseRunArgs reads flags for the run command.
// We use a small hand-written parser so you learn the ideas before
// introducing heavier CLI frameworks.
func parseRunArgs(args []string) (RunConfig, error) {
	cfg := RunConfig{
		Runs:     0, // required; 0 means "not set"
		Parallel: 1, // sequential by default
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		// Everything after "--" is the external command.
		if arg == "--" {
			cfg.Command = append([]string{}, args[i+1:]...)
			i = len(args)
			break
		}

		switch arg {
		case "--runs":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--runs requires a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return cfg, fmt.Errorf("--runs must be an integer: %w", err)
			}
			cfg.Runs = n
			i += 2
		case "--parallel":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--parallel requires a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return cfg, fmt.Errorf("--parallel must be an integer: %w", err)
			}
			cfg.Parallel = n
			i += 2
		default:
			return cfg, fmt.Errorf("unknown flag or argument %q (use -- before the test command)", arg)
		}
	}

	return cfg, nil
}

// ValidateRunConfig checks that a run configuration is safe to accept.
// Keeping validation separate makes it easy to unit-test without I/O.
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
	if len(cfg.Command) == 0 {
		return fmt.Errorf("missing test command after -- (example: -- python -m pytest)")
	}
	return nil
}
