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
	name          string
	appearances   int
	passed        int
	failed        int
	skipped       int
	errored       int
	durations     []int64
	passDurations []int64
	failDurations []int64
	predecessors  map[string]int // immediate prior test when this test failed/errored
}

// Analyze builds an AnalysisReport from a list of executions.
func Analyze(experimentID string, executions []model.Execution, th Thresholds) model.AnalysisReport {
	return AnalyzeExperiment(model.Experiment{ID: experimentID}, executions, th)
}

// AnalyzeExperiment is Analyze plus experiment metadata / Phase 6 variation signals.
func AnalyzeExperiment(exp model.Experiment, executions []model.Execution, th Thresholds) model.AnalysisReport {
	if th.MinOutcomes < 1 {
		th.MinOutcomes = 1
	}

	runsWithTests := 0
	runsWithOrder := 0
	seedSet := map[int64]bool{}
	var failingWorkers []int

	for _, e := range executions {
		if len(e.Tests) > 0 {
			runsWithTests++
		}
		if len(e.TestOrder) > 0 {
			runsWithOrder++
		}
		if e.Seed != 0 {
			seedSet[e.Seed] = true
		}
		if e.ExitCode != 0 || e.TimedOut || e.Error != "" {
			if e.WorkerID > 0 {
				failingWorkers = append(failingWorkers, e.WorkerID)
			}
		}
	}

	byName := map[string]*accum{}
	for _, e := range executions {
		order := e.TestOrder
		if len(order) == 0 {
			for _, tc := range e.Tests {
				order = append(order, tc.Name)
			}
		}
		indexOf := map[string]int{}
		for i, n := range order {
			indexOf[n] = i
		}

		seen := map[string]bool{}
		for _, tc := range e.Tests {
			a := byName[tc.Name]
			if a == nil {
				a = &accum{name: tc.Name, predecessors: map[string]int{}}
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
				recordPredecessor(a, order, indexOf, tc.Name)
			case model.TestSkip:
				a.skipped++
			case model.TestError:
				a.errored++
				a.failDurations = append(a.failDurations, tc.DurationMS)
				recordPredecessor(a, order, indexOf, tc.Name)
			}
		}
	}

	report := model.AnalysisReport{
		ExperimentID:  exp.ID,
		RunsAnalyzed:  len(executions),
		RunsWithTests: runsWithTests,
		UniqueTests:   len(byName),
		MinOutcomes:   th.MinOutcomes,
		OrderMode:     exp.OrderMode,
		Seed:          exp.Seed,
		Parallel:      exp.Parallel,
		Variation: &model.VariationSummary{
			DistinctSeeds:       len(seedSet),
			RunsWithTestOrder:   runsWithOrder,
			FailingRunWorkerIDs: uniqueInts(failingWorkers),
			OrderLimitations: []string{
				"JUnit XML order is best-effort; runners may not guarantee it matches true execution order.",
				"Random order requires a pytest plugin such as pytest-randomly (--randomly-seed).",
				"Parallel failure association is not proof of a concurrency bug.",
			},
		},
	}
	if runsWithTests == 0 {
		report.Notes = append(report.Notes,
			"No per-test results found. Analysis needs JUnit/pytest collection (or another parser).",
		)
	}
	if exp.OrderMode == model.OrderRandom {
		report.Notes = append(report.Notes,
			fmt.Sprintf("Order mode=random with base seed %d (run i uses seed+%d).", exp.Seed, 0),
			"Install pytest-randomly for --randomly-seed support, or random-order mode will fail at runtime.",
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

func recordPredecessor(a *accum, order []string, indexOf map[string]int, name string) {
	idx, ok := indexOf[name]
	if !ok || idx <= 0 || idx-1 >= len(order) {
		return
	}
	pred := order[idx-1]
	if pred == "" || pred == name {
		return
	}
	a.predecessors[pred]++
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
		TopPredecessors:      topPredecessors(a.predecessors, 3),
	}
	st.DurationMinMS, st.DurationMaxMS, st.DurationAvgMS = durationSummary(a.durations)
	_, _, st.PassedDurationAvgMS = durationSummary(a.passDurations)
	_, _, st.FailedDurationAvgMS = durationSummary(a.failDurations)

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
	if len(st.TopPredecessors) > 0 {
		p := st.TopPredecessors[0]
		ev = append(ev, fmt.Sprintf(
			"Most common immediate predecessor before failure: %s (%d times). Order evidence only.",
			p.Name, p.Count,
		))
	}
	return ev
}

func topPredecessors(m map[string]int, n int) []model.PredecessorStat {
	if len(m) == 0 {
		return nil
	}
	type pair struct {
		name  string
		count int
	}
	var ps []pair
	for k, v := range m {
		ps = append(ps, pair{k, v})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].count == ps[j].count {
			return ps[i].name < ps[j].name
		}
		return ps[i].count > ps[j].count
	})
	if len(ps) > n {
		ps = ps[:n]
	}
	out := make([]model.PredecessorStat, len(ps))
	for i, p := range ps {
		out[i] = model.PredecessorStat{Name: p.name, Count: p.count}
	}
	return out
}

func uniqueInts(in []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Ints(out)
	return out
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
