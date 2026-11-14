package engine

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunSuccess(t *testing.T) {
	argv := helperArgv(t, "ok")
	var stdout, stderr bytes.Buffer
	res := Run(context.Background(), Request{
		Argv:   argv,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if res.StartError != nil {
		t.Fatalf("start error: %v", res.StartError)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d stderr=%s", res.ExitCode, stderr.String())
	}
	if stdout.String() == "" {
		t.Fatal("expected stdout from helper")
	}
}

func TestRunFailureExitCode(t *testing.T) {
	argv := helperArgv(t, "fail")
	res := Run(context.Background(), Request{Argv: argv})
	if res.StartError != nil {
		t.Fatalf("start error: %v", res.StartError)
	}
	if res.ExitCode == 0 {
		t.Fatal("expected non-zero exit")
	}
}

func TestRunTimeout(t *testing.T) {
	argv := helperArgv(t, "sleep")
	res := Run(context.Background(), Request{
		Argv:    argv,
		Timeout: 500 * time.Millisecond,
	})
	if !res.TimedOut {
		t.Fatalf("expected timeout, got exit=%d startErr=%v dur=%s", res.ExitCode, res.StartError, res.Duration)
	}
}

func TestRunEmptyCommand(t *testing.T) {
	res := Run(context.Background(), Request{})
	if res.StartError == nil {
		t.Fatal("expected start error")
	}
}

func helperArgv(t *testing.T, mode string) []string {
	t.Helper()
	helper := filepath.Join("testdata", "helper", "main.go")
	// go run works on Windows and Unix; requires "go" on PATH.
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	return []string{"go", "run", helper, mode}
}

func TestOpenLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.log")
	f, err := OpenLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("hi"); err != nil {
		t.Fatal(err)
	}
	_ = runtime.GOOS
}
