// Package analysis computes deterministic flakiness statistics from executions.
//
// Important: this package reports evidence and classifications.
// It does not claim a root cause.
package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/edisshort/flakehunter/internal/model"
)

// Thresholds controls classification sensitivity.
type Thresholds struct {
	// MinOutcomes is the minimum decisive (pass/fail/error) results required
	// before we call a test stable, consistently failing, or flaky.
	MinOutcomes int
}

// DefaultThresholds returns MVP defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{MinOutcomes: 5}
}

type accum struct {
	name         string
	appearances  int
	passed       int
	failed       int
	skipped      int
	errored      int
	durations    []int64
	passDurations []int64
	failDurations []int64
}

// Analyze builds an AnalysisReport from a list of executions.
func Analyze(experimentID string, executions []model.Execution, th Thresholds) model.AnalysisReport {
	if th.MinOutcomes < 1 {
		th.MinOutcomes = 1
	}

	runsWithTests := 0
	for _, e := range executions {
		if len(e.Tests) > 0 {
			runsWithTests++
		}
	}

	byName := map[string]*accum{}
	for _, e := range executions {
		seen := map[string]bool{}
		for _, tc := range e.Tests {
			a := byName[tc.Name]
			if a == nil {
				a = &accum{name: tc.Name}
				byName[tc.Name] = a
			}
			if !seen[tc.Name] {
				a.appearances++
				seen[tc.Name] = true
			}
			a.durations = append(a.durations, tc.DurationMS)
			switch tc.Status {
			case model.TestPass:
				a.passed++
				a.passDurations = append(a.passDurations, tc.DurationMS)
			case model.TestFail:
				a.failed++
				a.failDurations = append(a.failDurations, tc.DurationMS)
			case model.TestSkip:
				a.skipped++
			case model.TestError:
				a.errored++
				a.failDurations = append(a.failDurations, tc.DurationMS)
			}
		}
	}

	report := model.AnalysisReport{
		ExperimentID:  experimentID,
		RunsAnalyzed:  len(executions),
		RunsWithTests: runsWithTests,
		UniqueTests:   len(byName),
		MinOutcomes:   th.MinOutcomes,
	}
	if runsWithTests == 0 {
		report.Notes = append(report.Notes,
			"No per-test results found. Analysis needs JUnit/pytest collection (or another parser).",
		)
	}

	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		st := classify(byName[name], runsWithTests, th)
		report.Tests = append(report.Tests, st)
		switch st.Classification {
		case model.ClassStablePassing:
			report.Counts.StablePassing++
		case model.ClassConsistentlyFailing:
			report.Counts.ConsistentlyFailing++
		case model.ClassFlaky:
			report.Counts.Flaky++
		case model.ClassInsufficient:
			report.Counts.InsufficientEvidence++
		case model.ClassInconsistentPresence:
			report.Counts.InconsistentPresence++
		}
	}
	return report
}

func classify(a *accum, runsWithTests int, th Thresholds) model.TestStats {
	decisive := a.passed + a.failed + a.errored
	failish := a.failed + a.errored
	var rate float64
	if decisive > 0 {
		rate = float64(failish) / float64(decisive)
	}

	st := model.TestStats{
		Name:                 a.name,
		Appearances:          a.appearances,
		ExpectedRuns:         runsWithTests,
		Passed:               a.passed,
		Failed:               a.failed,
		Skipped:              a.skipped,
		Errored:              a.errored,
		Decisive:             decisive,
		FailureRate:          round4(rate),
		InconsistentPresence: runsWithTests > 0 && a.appearances < runsWithTests,
	}
	st.DurationMinMS, st.DurationMaxMS, st.DurationAvgMS = durationSummary(a.durations)
	_, _, st.PassedDurationAvgMS = durationSummary(a.passDurations)
	_, _, st.FailedDurationAvgMS = durationSummary(a.failDurations)

	// Presence inconsistency is its own class when we otherwise lack a strong
	// flaky/stable signal, or when the test is missing from some collected runs.
	switch {
	case st.InconsistentPresence && decisive < th.MinOutcomes:
		st.Classification = model.ClassInconsistentPresence
	case decisive < th.MinOutcomes:
		st.Classification = model.ClassInsufficient
	case failish > 0 && a.passed > 0:
		st.Classification = model.ClassFlaky
	case failish > 0 && a.passed == 0:
		st.Classification = model.ClassConsistentlyFailing
	case st.InconsistentPresence:
		st.Classification = model.ClassInconsistentPresence
	default:
		st.Classification = model.ClassStablePassing
	}

	st.Evidence = buildEvidence(st)
	return st
}

func buildEvidence(st model.TestStats) []string {
	var ev []string
	ev = append(ev, fmt.Sprintf("Appeared in %d of %d runs with collected test results.", st.Appearances, st.ExpectedRuns))
	ev = append(ev, fmt.Sprintf("Decisive outcomes: %d (passed=%d failed=%d errored=%d skipped=%d).",
		st.Decisive, st.Passed, st.Failed, st.Errored, st.Skipped))
	if st.Decisive > 0 {
		ev = append(ev, fmt.Sprintf("Failure rate among decisive outcomes: %.2f%%.", st.FailureRate*100))
	}
	if st.InconsistentPresence {
		ev = append(ev, "Test was missing from some runs that produced JUnit results (inconsistent presence).")
	}
	if st.PassedDurationAvgMS > 0 && st.FailedDurationAvgMS > 0 &&
		st.FailedDurationAvgMS > st.PassedDurationAvgMS*1.5 {
		ev = append(ev, fmt.Sprintf(
			"Failed runs were slower on average (fail avg %.0fms vs pass avg %.0fms).",
			st.FailedDurationAvgMS, st.PassedDurationAvgMS,
		))
	}
	return ev
}

func durationSummary(vals []int64) (min, max int64, avg float64) {
	if len(vals) == 0 {
		return 0, 0, 0
	}
	min, max = vals[0], vals[0]
	var sum int64
	for _, v := range vals {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	avg = float64(sum) / float64(len(vals))
	return min, max, avg
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
