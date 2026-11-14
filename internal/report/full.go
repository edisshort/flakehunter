package report

import (
	"sort"
	"strings"
	"time"

	"github.com/edisshort/flakehunter/internal/model"
)

// BuildFullReport assembles the Phase 7 JSON report package.
func BuildFullReport(exp model.Experiment, analysis model.AnalysisReport, execs []model.Execution) model.FullReport {
	out := model.FullReport{
		GeneratedAt: time.Now().UTC(),
		Experiment:  exp,
		Analysis:    analysis,
	}
	for _, e := range execs {
		sum := model.ExecutionSummary{
			RunIndex:     e.RunIndex,
			ExitCode:     e.ExitCode,
			TimedOut:     e.TimedOut,
			DurationMS:   e.DurationMS,
			WorkerID:     e.WorkerID,
			Seed:         e.Seed,
			OrderMode:    e.OrderMode,
			TestCount:    len(e.Tests),
			CollectError: e.CollectError,
			Error:        e.Error,
		}
		for _, tc := range e.Tests {
			if tc.Status == model.TestFail || tc.Status == model.TestError {
				sum.FailedNames = append(sum.FailedNames, tc.Name)
			}
		}
		out.Executions = append(out.Executions, sum)
	}
	sort.Slice(out.Executions, func(i, j int) bool {
		return out.Executions[i].RunIndex < out.Executions[j].RunIndex
	})
	return out
}

// InspectTest builds a per-test history across executions.
// query may be an exact name or a case-insensitive substring.
func InspectTest(expID string, analysis model.AnalysisReport, execs []model.Execution, query string) model.TestInspection {
	query = strings.TrimSpace(query)
	insp := model.TestInspection{
		ExperimentID: expID,
		TestName:     query,
	}
	if query == "" {
		insp.Notes = append(insp.Notes, "Empty test name.")
		return insp
	}

	// Prefer exact match among known stats; else substring.
	var matched string
	for _, t := range analysis.Tests {
		if t.Name == query {
			matched = t.Name
			st := t
			insp.Stats = &st
			insp.MatchedAs = "exact"
			break
		}
	}
	if matched == "" {
		q := strings.ToLower(query)
		var candidates []string
		for _, t := range analysis.Tests {
			if strings.Contains(strings.ToLower(t.Name), q) {
				candidates = append(candidates, t.Name)
			}
		}
		sort.Strings(candidates)
		if len(candidates) == 1 {
			matched = candidates[0]
			insp.MatchedAs = "substring"
			for _, t := range analysis.Tests {
				if t.Name == matched {
					st := t
					insp.Stats = &st
					break
				}
			}
		} else if len(candidates) > 1 {
			insp.Notes = append(insp.Notes, "Multiple matches; be more specific:")
			insp.Notes = append(insp.Notes, candidates...)
			return insp
		}
	}
	if matched == "" {
		// Still walk history for substring in raw executions.
		matched = query
		insp.MatchedAs = "query"
		insp.Notes = append(insp.Notes, "No aggregated stats for this name; showing raw history matches if any.")
	}
	insp.TestName = matched

	for _, e := range execs {
		obs := model.TestRunObservation{
			RunIndex: e.RunIndex,
			Status:   "missing",
			WorkerID: e.WorkerID,
			Seed:     e.Seed,
		}
		order := e.TestOrder
		if len(order) == 0 {
			for _, tc := range e.Tests {
				order = append(order, tc.Name)
			}
		}
		idx := -1
		for i, n := range order {
			if n == matched || (insp.MatchedAs == "query" && strings.Contains(strings.ToLower(n), strings.ToLower(query))) {
				idx = i
				if n != matched && insp.MatchedAs == "query" {
					matched = n
					insp.TestName = n
				}
				break
			}
		}
		found := false
		for _, tc := range e.Tests {
			nameOK := tc.Name == matched || (insp.MatchedAs == "query" && strings.Contains(strings.ToLower(tc.Name), strings.ToLower(query)))
			if !nameOK {
				continue
			}
			found = true
			obs.Status = tc.Status
			obs.DurationMS = tc.DurationMS
			obs.Message = tc.Message
			if idx > 0 {
				obs.Predecessor = order[idx-1]
			}
			if tc.Name != matched {
				matched = tc.Name
				insp.TestName = matched
			}
			break
		}
		if !found {
			obs.Status = "missing"
		}
		insp.History = append(insp.History, obs)
	}
	sort.Slice(insp.History, func(i, j int) bool {
		return insp.History[i].RunIndex < insp.History[j].RunIndex
	})
	return insp
}
