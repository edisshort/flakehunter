// Package cli implements FlakeHunter's command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/edisshort/flakehunter/internal/analysis"
	"github.com/edisshort/flakehunter/internal/experiment"
	"github.com/edisshort/flakehunter/internal/model"
	"github.com/edisshort/flakehunter/internal/report"
	"github.com/edisshort/flakehunter/internal/store"
)

// Version is reported by the "version" command.
const Version = "0.9.1-vitest"

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
	OrderMode   string
	Seed        int64
	SeedSet     bool
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
		return cmdList(args[1:], stdout, stderr)
	case "report":
		return cmdReport(args[1:], stdout, stderr)
	case "inspect":
		return cmdInspect(args[1:], stdout, stderr)
	case "compare":
		return cmdCompare(args[1:], stdout, stderr)
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
	seed := cfg.Seed
	if !cfg.SeedSet {
		seed = 0 // let experiment auto-generate when random
	}
	_, err := runner.Run(ctx, experiment.Config{
		Command:     cfg.Command,
		WorkingDir:  cfg.WorkingDir,
		Runs:        cfg.Runs,
		Parallel:    cfg.Parallel,
		Timeout:     cfg.Timeout,
		MinOutcomes: cfg.MinOutcomes,
		OrderMode:   cfg.OrderMode,
		Seed:        seed,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitRuntime
	}
	return ExitOK
}

func cmdList(args []string, stdout, stderr io.Writer) int {
	root, _, err := parseWorkdirOnly(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}
	s := store.New(root)
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
		order := exp.OrderMode
		if order == "" {
			order = model.OrderDefault
		}
		fmt.Fprintf(stdout, "  %s  status=%s runs=%d parallel=%d order=%s seed=%d cmd=%s\n",
			id, exp.Status, exp.RunsRequested, exp.Parallel, order, exp.Seed, strings.Join(exp.Command, " "))
	}
	return ExitOK
}

func cmdReport(args []string, stdout, stderr io.Writer) int {
	root, rest, err := parseWorkdirOnly(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}
	args = rest
	s := store.New(root)
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
	rep := analysis.AnalyzeExperiment(exp, execs, th)
	if err := s.WriteAnalysis(id, rep); err != nil {
		fmt.Fprintf(stderr, "error: save analysis: %v\n", err)
		return ExitRuntime
	}
	full := report.BuildFullReport(exp, rep, execs)
	if err := s.WriteFullReport(id, full); err != nil {
		fmt.Fprintf(stderr, "error: save report: %v\n", err)
		return ExitRuntime
	}
	report.WriteFullTerminal(stdout, full)
	fmt.Fprintf(stdout, "Saved analysis: %s\n", filepath.Join(s.ExperimentDir(id), "analysis.json"))
	fmt.Fprintf(stdout, "Saved report:   %s\n", filepath.Join(s.ExperimentDir(id), "report.json"))
	return ExitOK
}

func cmdInspect(args []string, stdout, stderr io.Writer) int {
	root, rest, err := parseWorkdirOnly(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}
	args = rest
	if len(args) == 0 {
		fmt.Fprintf(stderr, "error: usage: flakehunter inspect <test-name> [--experiment <id>] [--workdir <path>]\n")
		return ExitUsage
	}
	s := store.New(root)
	id := ""
	var nameParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--experiment", "-e":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "error: %s requires an id\n", args[i])
				return ExitUsage
			}
			id = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "error: unknown flag %q\n", args[i])
				return ExitUsage
			}
			nameParts = append(nameParts, args[i])
		}
	}
	query := strings.Join(nameParts, " ")
	if query == "" {
		fmt.Fprintf(stderr, "error: missing test name\n")
		return ExitUsage
	}

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
	analysisRep, err := s.ReadAnalysis(id)
	if err != nil {
		// Recompute if missing.
		analysisRep = analysis.AnalyzeExperiment(exp, execs, analysis.DefaultThresholds())
	}
	insp := report.InspectTest(id, analysisRep, execs, query)
	report.WriteInspection(stdout, insp)

	safe := sanitizeFileName(insp.TestName)
	if err := s.WriteInspection(id, safe, insp); err != nil {
		fmt.Fprintf(stderr, "warning: could not save inspection: %v\n", err)
	} else {
		fmt.Fprintf(stdout, "\nSaved inspection: %s\n",
			filepath.Join(s.ExperimentDir(id), "inspections", safe+".json"))
	}
	return ExitOK
}

func sanitizeFileName(name string) string {
	repl := strings.NewReplacer(`\`, "_", `/`, "_", `:`, "_", `*`, "_", `?`, "_", `"`, "_", `<`, "_", `>`, "_", `|`, "_", ` `, "_")
	out := repl.Replace(name)
	if len(out) > 120 {
		out = out[:120]
	}
	if out == "" {
		return "test"
	}
	return out
}

func cmdCompare(args []string, stdout, stderr io.Writer) int {
	root, rest, err := parseWorkdirOnly(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return ExitUsage
	}
	args = rest
	if len(args) < 2 {
		fmt.Fprintf(stderr, "error: usage: flakehunter compare <left-id> <right-id> [--workdir <path>]\n")
		return ExitUsage
	}
	leftID, rightID := args[0], args[1]
	minOutcomes := 0
	for i := 2; i < len(args); i++ {
		if args[i] == "--min-outcomes" {
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
			continue
		}
		fmt.Fprintf(stderr, "error: unexpected argument %q\n", args[i])
		return ExitUsage
	}

	s := store.New(root)
	leftExp, err := s.ReadExperiment(leftID)
	if err != nil {
		fmt.Fprintf(stderr, "error: left experiment: %v\n", err)
		return ExitRuntime
	}
	rightExp, err := s.ReadExperiment(rightID)
	if err != nil {
		fmt.Fprintf(stderr, "error: right experiment: %v\n", err)
		return ExitRuntime
	}
	leftExecs, err := s.LoadExecutions(leftID, leftExp.RunsRequested)
	if err != nil {
		fmt.Fprintf(stderr, "error: left executions: %v\n", err)
		return ExitRuntime
	}
	rightExecs, err := s.LoadExecutions(rightID, rightExp.RunsRequested)
	if err != nil {
		fmt.Fprintf(stderr, "error: right executions: %v\n", err)
		return ExitRuntime
	}
	th := analysis.DefaultThresholds()
	if minOutcomes > 0 {
		th.MinOutcomes = minOutcomes
	}
	rep := analysis.Compare(leftExp, rightExp, leftExecs, rightExecs, th)
	report.WriteCompare(stdout, rep)

	outPath := filepath.Join(s.BaseDir(), "compare-"+leftID+"-vs-"+rightID+".json")
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "warning: could not encode compare json: %v\n", err)
		return ExitOK
	}
	if err := os.MkdirAll(s.BaseDir(), 0o755); err == nil {
		_ = os.WriteFile(outPath, append(data, '\n'), 0o644)
		fmt.Fprintf(stdout, "Saved compare: %s\n", outPath)
	}
	return ExitOK
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `FlakeHunter — detect and investigate flaky automated tests

Usage:
  flakehunter <command> [flags]

Commands:
  run       Run a repeated test experiment (sequential or parallel)
  report    Analyze an experiment (terminal + analysis.json + report.json)
  inspect   Deep-dive one test's history and evidence
  compare   Compare two experiments (e.g. sequential vs parallel)
  list      List saved experiments under .flakehunter/
  version   Print the FlakeHunter version
  help      Show this help message

Examples:
  flakehunter run --runs 30 -- python -m pytest
  flakehunter run --runs 30 --parallel 4 -- python -m pytest
  flakehunter run --runs 20 --order random --seed 42 -- python -m pytest
  flakehunter report
  flakehunter inspect test_coin_flip
  flakehunter compare <seq-id> <par-id>
  flakehunter list

Notes:
  - pytest: unique --junitxml per run.
  - vitest / npm test: unique JUnit --reporter/--outputFile per run.
  - --order random needs pytest-randomly (--randomly-seed).
  - JUnit order is best-effort evidence, not guaranteed execution order.
  - Classifications and comparisons are evidence, not root-cause claims.
  - Install on another machine: see INSTALL.md (go install or release binary).
  - Example flaky suite: testdata/flaky_suite (Phase 8).

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
  --order <mode>       default | random (random uses pytest-randomly)
  --seed <n>           Base seed for random order (run i uses seed+i)

The "--" separator marks the start of the external test command.

`)
}

func parseRunArgs(args []string) (RunConfig, error) {
	cfg := RunConfig{
		Runs:      0,
		Parallel:  1,
		OrderMode: model.OrderDefault,
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
		case "--order":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--order requires default or random")
			}
			mode := strings.ToLower(args[i+1])
			if mode != model.OrderDefault && mode != model.OrderRandom {
				return cfg, fmt.Errorf("--order must be %q or %q", model.OrderDefault, model.OrderRandom)
			}
			cfg.OrderMode = mode
			i += 2
		case "--seed":
			if i+1 >= len(args) {
				return cfg, fmt.Errorf("--seed requires an integer")
			}
			n, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return cfg, fmt.Errorf("--seed must be an integer: %w", err)
			}
			cfg.Seed = n
			cfg.SeedSet = true
			i += 2
		default:
			return cfg, fmt.Errorf("unknown flag or argument %q (use -- before the test command)", arg)
		}
	}

	return cfg, nil
}

// parseWorkdirOnly extracts an optional --workdir flag from command args.
func parseWorkdirOnly(args []string) (root string, rest []string, err error) {
	root = "."
	for i := 0; i < len(args); i++ {
		if args[i] == "--workdir" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--workdir requires a path")
			}
			root = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	return root, rest, nil
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
	if cfg.OrderMode != "" && cfg.OrderMode != model.OrderDefault && cfg.OrderMode != model.OrderRandom {
		return fmt.Errorf("--order must be default or random")
	}
	if cfg.SeedSet && cfg.OrderMode != model.OrderRandom {
		return fmt.Errorf("--seed is only valid with --order random")
	}
	if len(cfg.Command) == 0 {
		return fmt.Errorf("missing test command after -- (example: -- python -m pytest)")
	}
	return nil
}
