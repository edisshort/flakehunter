// Package model holds shared experiment data types.
// Keeping these types in one place lets storage, execution, and analysis
// speak the same language without importing each other in circles.
package model

import "time"

// Experiment statuses.
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Test statuses from a single execution.
const (
	TestPass  = "pass"
	TestFail  = "fail"
	TestSkip  = "skip"
	TestError = "error"
)

// Experiment is one multi-run trial (metadata only).
type Experiment struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	Command       []string  `json:"command"`
	WorkingDir    string    `json:"working_dir"`
	RunsRequested int       `json:"runs_requested"`
	Parallel      int       `json:"parallel"`
	TimeoutMS     int64     `json:"timeout_ms"` // 0 = no timeout
	Status        string    `json:"status"`
	GoVersion     string    `json:"go_version"`
	GOOS          string    `json:"goos"`
	GOARCH        string    `json:"goarch"`
	Note          string    `json:"note,omitempty"`
}

// Execution is one run of the external test command.
type Execution struct {
	ExperimentID string    `json:"experiment_id"`
	RunIndex     int       `json:"run_index"` // 1-based for humans
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	DurationMS   int64     `json:"duration_ms"`
	ExitCode     int       `json:"exit_code"`
	TimedOut     bool      `json:"timed_out"`
	StdoutPath   string    `json:"stdout_path"`
	StderrPath   string    `json:"stderr_path"`
	JUnitPath    string    `json:"junit_path,omitempty"`
	Error        string    `json:"error,omitempty"`
	Tests        []TestCase `json:"tests,omitempty"`
	CollectError string    `json:"collect_error,omitempty"`
}

// TestCase is one test outcome inside one execution.
type TestCase struct {
	Name       string `json:"name"`
	Classname  string `json:"classname,omitempty"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Message    string `json:"message,omitempty"`
}
