package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestValidateRunConfigOK(t *testing.T) {
	cfg := RunConfig{
		Runs:     30,
		Parallel: 4,
		Command:  []string{"python", "-m", "pytest"},
	}
	if err := ValidateRunConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateRunConfigRejectsMissingCommand(t *testing.T) {
	cfg := RunConfig{Runs: 5, Parallel: 1, Command: nil}
	err := ValidateRunConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "missing test command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRunConfigRejectsBadRuns(t *testing.T) {
	cfg := RunConfig{Runs: 0, Parallel: 1, Command: []string{"pytest"}}
	if err := ValidateRunConfig(cfg); err == nil {
		t.Fatal("expected error for runs < 1")
	}
}

func TestValidateRunConfigRejectsParallelGreaterThanRuns(t *testing.T) {
	cfg := RunConfig{Runs: 2, Parallel: 5, Command: []string{"pytest"}}
	if err := ValidateRunConfig(cfg); err == nil {
		t.Fatal("expected error when parallel > runs")
	}
}

func TestParseRunArgs(t *testing.T) {
	cfg, err := parseRunArgs([]string{
		"--runs", "10",
		"--parallel", "2",
		"--timeout", "30s",
		"--workdir", "C:\\tmp",
		"--order", "random",
		"--seed", "42",
		"--", "python", "-m", "pytest",
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.Runs != 10 || cfg.Parallel != 2 {
		t.Fatalf("got runs=%d parallel=%d", cfg.Runs, cfg.Parallel)
	}
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("timeout=%s", cfg.Timeout)
	}
	if cfg.WorkingDir != `C:\tmp` {
		t.Fatalf("workdir=%q", cfg.WorkingDir)
	}
	if cfg.OrderMode != "random" || cfg.Seed != 42 || !cfg.SeedSet {
		t.Fatalf("order/seed=%s/%d set=%v", cfg.OrderMode, cfg.Seed, cfg.SeedSet)
	}
	if len(cfg.Command) != 3 || cfg.Command[0] != "python" {
		t.Fatalf("unexpected command: %#v", cfg.Command)
	}
}

func TestValidateSeedRequiresRandom(t *testing.T) {
	cfg := RunConfig{
		Runs: 2, Parallel: 1, SeedSet: true, Seed: 1,
		OrderMode: "default", Command: []string{"pytest"},
	}
	if err := ValidateRunConfig(cfg); err == nil {
		t.Fatal("expected seed/order validation error")
	}
}

func TestRunHelp(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"help"}, &out, &out)
	if code != ExitOK {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "FlakeHunter") {
		t.Fatalf("help missing brand text: %q", out.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"version"}, &out, &out)
	if code != ExitOK {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), Version) {
		t.Fatalf("version output missing version: %q", out.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"explode"}, &out, &errBuf)
	if code != ExitUsage {
		t.Fatalf("expected usage exit, got %d", code)
	}
}

func TestRunValidationError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"run", "--runs", "0", "--", "pytest"}, &out, &errBuf)
	if code != ExitValidation {
		t.Fatalf("expected validation exit, got %d stderr=%s", code, errBuf.String())
	}
}
