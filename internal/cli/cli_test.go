package cli

import (
	"bytes"
	"strings"
	"testing"
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
	cfg, err := parseRunArgs([]string{"--runs", "10", "--parallel", "2", "--", "python", "-m", "pytest"})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if cfg.Runs != 10 || cfg.Parallel != 2 {
		t.Fatalf("got runs=%d parallel=%d", cfg.Runs, cfg.Parallel)
	}
	if len(cfg.Command) != 3 || cfg.Command[0] != "python" {
		t.Fatalf("unexpected command: %#v", cfg.Command)
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

func TestRunStubPrintsPlan(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"run", "--runs", "5", "--", "python", "-m", "pytest"}, &out, &errBuf)
	if code != ExitOK {
		t.Fatalf("exit %d stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "runs:     5") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"explode"}, &out, &errBuf)
	if code != ExitUsage {
		t.Fatalf("expected usage exit, got %d", code)
	}
}
