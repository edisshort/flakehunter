package model

import "time"

// FullReport is the Phase 7 reproducible JSON artifact for one experiment.
type FullReport struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Experiment  Experiment        `json:"experiment"`
	Analysis    AnalysisReport    `json:"analysis"`
	Executions  []ExecutionSummary `json:"executions"`
}

// ExecutionSummary is a compact per-run view for reports (not full log paths only).
type ExecutionSummary struct {
	RunIndex     int      `json:"run_index"`
	ExitCode     int      `json:"exit_code"`
	TimedOut     bool     `json:"timed_out"`
	DurationMS   int64    `json:"duration_ms"`
	WorkerID     int      `json:"worker_id,omitempty"`
	Seed         int64    `json:"seed,omitempty"`
	OrderMode    string   `json:"order_mode,omitempty"`
	TestCount    int      `json:"test_count"`
	FailedNames  []string `json:"failed_names,omitempty"`
	CollectError string   `json:"collect_error,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// TestInspection is the Phase 7 per-test deep dive.
type TestInspection struct {
	ExperimentID   string              `json:"experiment_id"`
	TestName       string              `json:"test_name"`
	Stats          *TestStats          `json:"stats,omitempty"`
	MatchedAs      string              `json:"matched_as,omitempty"` // exact or substring match used
	History        []TestRunObservation `json:"history"`
	Notes          []string            `json:"notes,omitempty"`
}

// TestRunObservation is one execution's outcome for a named test.
type TestRunObservation struct {
	RunIndex   int    `json:"run_index"`
	Status     string `json:"status"` // pass/fail/skip/error/missing
	DurationMS int64  `json:"duration_ms,omitempty"`
	Message    string `json:"message,omitempty"`
	WorkerID   int    `json:"worker_id,omitempty"`
	Seed       int64  `json:"seed,omitempty"`
	Predecessor string `json:"predecessor,omitempty"`
}
