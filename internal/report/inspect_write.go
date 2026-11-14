package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/edisshort/flakehunter/internal/model"
)

// WriteInspection prints a per-test deep dive.
func WriteInspection(w io.Writer, insp model.TestInspection) {
	fmt.Fprintln(w, "FlakeHunter Test Inspection")
	fmt.Fprintln(w, strings.Repeat("=", 28))
	fmt.Fprintf(w, "Experiment: %s\n", insp.ExperimentID)
	fmt.Fprintf(w, "Test:       %s\n", insp.TestName)
	if insp.MatchedAs != "" {
		fmt.Fprintf(w, "Matched as: %s\n", insp.MatchedAs)
	}
	fmt.Fprintln(w)

	for _, n := range insp.Notes {
		fmt.Fprintf(w, "Note: %s\n", n)
	}
	if len(insp.Notes) > 0 {
		fmt.Fprintln(w)
	}

	if insp.Stats != nil {
		st := insp.Stats
		fmt.Fprintf(w, "Classification: %s\n", st.Classification)
		fmt.Fprintf(w, "Appearances:    %d / %d\n", st.Appearances, st.ExpectedRuns)
		fmt.Fprintf(w, "Passed/Failed/Errored/Skipped: %d/%d/%d/%d\n",
			st.Passed, st.Failed, st.Errored, st.Skipped)
		fmt.Fprintf(w, "Failure rate:   %.2f%%\n", st.FailureRate*100)
		fmt.Fprintf(w, "Duration ms:    min=%d avg=%.1f max=%d\n",
			st.DurationMinMS, st.DurationAvgMS, st.DurationMaxMS)
		if len(st.TopPredecessors) > 0 {
			fmt.Fprintln(w, "Top predecessors before failure:")
			for _, p := range st.TopPredecessors {
				fmt.Fprintf(w, "  - %s (%d)\n", p.Name, p.Count)
			}
		}
		if len(st.Evidence) > 0 {
			fmt.Fprintln(w, "Evidence:")
			for _, e := range st.Evidence {
				fmt.Fprintf(w, "  - %s\n", e)
			}
		}
		fmt.Fprintln(w, "(Evidence only — not a proven root cause.)")
		fmt.Fprintln(w)
	}

	if len(insp.History) == 0 {
		fmt.Fprintln(w, "No execution history.")
		return
	}
	fmt.Fprintln(w, "Execution history:")
	for _, h := range insp.History {
		line := fmt.Sprintf("  run %04d  status=%-7s  dur=%dms", h.RunIndex, h.Status, h.DurationMS)
		if h.WorkerID > 0 {
			line += fmt.Sprintf("  worker=%d", h.WorkerID)
		}
		if h.Seed != 0 {
			line += fmt.Sprintf("  seed=%d", h.Seed)
		}
		if h.Predecessor != "" {
			line += fmt.Sprintf("  after=%s", h.Predecessor)
		}
		fmt.Fprintln(w, line)
		if h.Message != "" {
			fmt.Fprintf(w, "           message: %s\n", truncate(h.Message, 160))
		}
	}
}

// WriteExecutionHistory prints a compact run table for the full report.
func WriteExecutionHistory(w io.Writer, full model.FullReport) {
	fmt.Fprintln(w, "Execution history")
	fmt.Fprintln(w, strings.Repeat("-", 17))
	for _, e := range full.Executions {
		line := fmt.Sprintf("  run %04d  exit=%d  dur=%dms  tests=%d",
			e.RunIndex, e.ExitCode, e.DurationMS, e.TestCount)
		if e.TimedOut {
			line += "  timed_out"
		}
		if e.WorkerID > 0 {
			line += fmt.Sprintf("  worker=%d", e.WorkerID)
		}
		if len(e.FailedNames) > 0 {
			line += fmt.Sprintf("  failed=%d", len(e.FailedNames))
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
