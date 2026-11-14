package model

// Classification labels for per-test flakiness analysis.
const (
	ClassStablePassing        = "stable_passing"
	ClassConsistentlyFailing  = "consistently_failing"
	ClassFlaky                = "flaky"
	ClassInsufficient         = "insufficient_evidence"
	ClassInconsistentPresence = "inconsistent_presence"
)

// TestStats is derived statistics for one test across an experiment.
type TestStats struct {
	Name                 string            `json:"name"`
	Appearances          int               `json:"appearances"`
	ExpectedRuns         int               `json:"expected_runs"`
	Passed               int               `json:"passed"`
	Failed               int               `json:"failed"`
	Skipped              int               `json:"skipped"`
	Errored              int               `json:"errored"`
	Decisive             int               `json:"decisive"` // pass + fail + error
	FailureRate          float64           `json:"failure_rate"`
	DurationMinMS        int64             `json:"duration_min_ms"`
	DurationMaxMS        int64             `json:"duration_max_ms"`
	DurationAvgMS        float64           `json:"duration_avg_ms"`
	FailedDurationAvgMS  float64           `json:"failed_duration_avg_ms,omitempty"`
	PassedDurationAvgMS  float64           `json:"passed_duration_avg_ms,omitempty"`
	Classification       string            `json:"classification"`
	InconsistentPresence bool              `json:"inconsistent_presence"`
	Evidence             []string          `json:"evidence,omitempty"`
	TopPredecessors      []PredecessorStat `json:"top_predecessors,omitempty"`
}

// PredecessorStat counts how often another test appeared immediately before a failure.
type PredecessorStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// AnalysisReport is the full analysis output for one experiment.
type AnalysisReport struct {
	ExperimentID  string            `json:"experiment_id"`
	RunsAnalyzed  int               `json:"runs_analyzed"`
	RunsWithTests int               `json:"runs_with_tests"`
	UniqueTests   int               `json:"unique_tests"`
	MinOutcomes   int               `json:"min_outcomes"`
	OrderMode     string            `json:"order_mode,omitempty"`
	Seed          int64             `json:"seed,omitempty"`
	Parallel      int               `json:"parallel,omitempty"`
	Counts        ClassCounts       `json:"counts"`
	Tests         []TestStats       `json:"tests"`
	Variation     *VariationSummary `json:"variation,omitempty"`
	Notes         []string          `json:"notes,omitempty"`
}

// VariationSummary captures Phase 6 order / parallel signals inside one experiment.
type VariationSummary struct {
	DistinctSeeds       int      `json:"distinct_seeds"`
	RunsWithTestOrder   int      `json:"runs_with_test_order"`
	OrderLimitations    []string `json:"order_limitations,omitempty"`
	FailingRunWorkerIDs []int    `json:"failing_run_worker_ids,omitempty"`
}

// CompareReport contrasts two experiments (e.g. sequential vs parallel, or different seeds).
type CompareReport struct {
	LeftID    string           `json:"left_id"`
	RightID   string           `json:"right_id"`
	LeftMeta  CompareMeta      `json:"left_meta"`
	RightMeta CompareMeta      `json:"right_meta"`
	Tests     []CompareTestRow `json:"tests"`
	Notes     []string         `json:"notes,omitempty"`
}

// CompareMeta is the experiment configuration slice used in comparisons.
type CompareMeta struct {
	Parallel  int    `json:"parallel"`
	OrderMode string `json:"order_mode"`
	Seed      int64  `json:"seed,omitempty"`
	Runs      int    `json:"runs"`
}

// CompareTestRow is one test's failure-rate contrast across two experiments.
type CompareTestRow struct {
	Name             string  `json:"name"`
	LeftFailureRate  float64 `json:"left_failure_rate"`
	RightFailureRate float64 `json:"right_failure_rate"`
	LeftDecisive     int     `json:"left_decisive"`
	RightDecisive    int     `json:"right_decisive"`
	Delta            float64 `json:"delta"` // right - left
	Evidence         string  `json:"evidence,omitempty"`
}

// ClassCounts summarizes how many tests fell into each bucket.
type ClassCounts struct {
	StablePassing        int `json:"stable_passing"`
	ConsistentlyFailing  int `json:"consistently_failing"`
	Flaky                int `json:"flaky"`
	InsufficientEvidence int `json:"insufficient_evidence"`
	InconsistentPresence int `json:"inconsistent_presence"`
}
