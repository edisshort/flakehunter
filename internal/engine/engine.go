// Package engine runs external commands safely for FlakeHunter experiments.
//
// Core ideas (Phase 2):
//   - os/exec starts another program (pytest, a helper script, etc.)
//   - context.Context carries cancellation and timeouts
//   - We capture stdout/stderr to files and record exit code + duration
package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Request describes one subprocess launch.
type Request struct {
	// Argv is the full command: Argv[0] is the program, the rest are args.
	Argv []string
	// Dir is the working directory (empty = inherit current).
	Dir string
	// Env is the full environment (nil = inherit process env).
	Env []string
	// Timeout stops the process if it runs too long (0 = no timeout).
	Timeout time.Duration
	// Stdout and Stderr receive process output (usually file writers).
	Stdout io.Writer
	Stderr io.Writer
}

// Result is what happened after the process finished or was stopped.
type Result struct {
	ExitCode   int
	TimedOut   bool
	Duration   time.Duration
	StartError error // non-nil if the process never started
}

// Run starts the command and waits until it exits or the context/timeout fires.
func Run(parent context.Context, req Request) Result {
	started := time.Now()
	out := Result{ExitCode: -1}

	if len(req.Argv) == 0 {
		out.StartError = errors.New("empty command")
		out.Duration = time.Since(started)
		return out
	}

	ctx := parent
	var cancel context.CancelFunc
	if req.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, req.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = req.Dir
	if req.Env != nil {
		cmd.Env = req.Env
	}
	cmd.Stdout = req.Stdout
	cmd.Stderr = req.Stderr

	err := cmd.Run()
	out.Duration = time.Since(started)

	if err == nil {
		out.ExitCode = 0
		return out
	}

	// Did we hit the deadline?
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		out.TimedOut = true
		out.ExitCode = -1
		return out
	}

	// Process started but exited non-zero.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.ExitCode = exitErr.ExitCode()
		return out
	}

	// Could not start (missing binary, bad path, etc.).
	out.StartError = fmt.Errorf("start command: %w", err)
	return out
}

// OpenLogFile creates/truncates a log file for stdout or stderr capture.
func OpenLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
}
