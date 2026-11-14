package model

// Classification labels for per-test flakiness analysis.
const (
	ClassStablePassing       = "stable_passing"
	ClassConsistentlyFailing = "consistently_failing"
	ClassFlaky               = "flaky"
	ClassInsufficient        = "insufficient_evidence"
	ClassInconsistentPresence = "inconsistent_presence"
)

// TestStats is derived statistics for one test across an experiment.
type TestStats struct {
	Name                  string   `json:"name"`
	Appearances           int      `json:"appearances"`
	ExpectedRuns          int      `json:"expected_runs"`
	Passed                int      `json:"passed"`
	Failed                int      `json:"failed"`
	Skipped               int      `json:"skipped"`
	Errored               int      `json:"errored"`
	Decisive              int      `json:"decisive"` // pass + fail + error
	FailureRate           float64  `json:"failure_rate"`
	DurationMinMS         int64    `json:"duration_min_ms"`
	DurationMaxMS         int64    `json:"duration_max_ms"`
	DurationAvgMS         float64  `json:"duration_avg_ms"`
	FailedDurationAvgMS   float64  `json:"failed_duration_avg_ms,omitempty"`
	PassedDurationAvgMS   float64  `json:"passed_duration_avg_ms,omitempty"`
	Classification        string   `json:"classification"`
	InconsistentPresence  bool     `json:"inconsistent_presence"`
	Evidence              []string `json:"evidence,omitempty"`
}

// AnalysisReport is the full Phase 4 output for one experiment.
type AnalysisReport struct {
	ExperimentID   string      `json:"experiment_id"`
	RunsAnalyzed   int         `json:"runs_analyzed"`
	RunsWithTests  int         `json:"runs_with_tests"`
	UniqueTests    int         `json:"unique_tests"`
	MinOutcomes    int         `json:"min_outcomes"`
	Counts         ClassCounts `json:"counts"`
	Tests          []TestStats `json:"tests"`
	Notes          []string    `json:"notes,omitempty"`
}

// ClassCounts summarizes how many tests fell into each bucket.
type ClassCounts struct {
	StablePassing        int `json:"stable_passing"`
	ConsistentlyFailing  int `json:"consistently_failing"`
	Flaky                int `json:"flaky"`
	InsufficientEvidence int `json:"insufficient_evidence"`
	InconsistentPresence int `json:"inconsistent_presence"`
}
