// Package report formats analysis results for the terminal and JSON files.
package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/edisshort/flakehunter/internal/model"
)

// WriteTerminal prints a human-readable summary.
func WriteTerminal(w io.Writer, rep model.AnalysisReport) {
	fmt.Fprintln(w, "FlakeHunter Experiment Report")
	fmt.Fprintln(w, strings.Repeat("=", 32))
	fmt.Fprintf(w, "Experiment:              %s\n", rep.ExperimentID)
	fmt.Fprintf(w, "Executions analyzed:     %d\n", rep.RunsAnalyzed)
	fmt.Fprintf(w, "Runs with test results:  %d\n", rep.RunsWithTests)
	fmt.Fprintf(w, "Unique tests:            %d\n", rep.UniqueTests)
	fmt.Fprintf(w, "Min decisive outcomes:   %d\n\n", rep.MinOutcomes)

	fmt.Fprintf(w, "Stable passing:          %d\n", rep.Counts.StablePassing)
	fmt.Fprintf(w, "Flaky:                   %d\n", rep.Counts.Flaky)
	fmt.Fprintf(w, "Consistently failing:    %d\n", rep.Counts.ConsistentlyFailing)
	fmt.Fprintf(w, "Insufficient evidence:   %d\n", rep.Counts.InsufficientEvidence)
	fmt.Fprintf(w, "Inconsistent presence:   %d\n\n", rep.Counts.InconsistentPresence)

	for _, note := range rep.Notes {
		fmt.Fprintf(w, "Note: %s\n", note)
	}
	if len(rep.Notes) > 0 {
		fmt.Fprintln(w)
	}

	printSection(w, "Flaky tests", rep, model.ClassFlaky)
	printSection(w, "Consistently failing tests", rep, model.ClassConsistentlyFailing)
	printSection(w, "Inconsistent presence", rep, model.ClassInconsistentPresence)
}

func printSection(w io.Writer, title string, rep model.AnalysisReport, class string) {
	var items []model.TestStats
	for _, t := range rep.Tests {
		if t.Classification == class {
			items = append(items, t)
		}
	}
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "%s (%d)\n", title, len(items))
	fmt.Fprintln(w, strings.Repeat("-", len(title)+6))
	for _, t := range items {
		fmt.Fprintf(w, "\nName: %s\n", t.Name)
		fmt.Fprintf(w, "  Appearances: %d / %d\n", t.Appearances, t.ExpectedRuns)
		fmt.Fprintf(w, "  Passed: %d  Failed: %d  Errored: %d  Skipped: %d\n",
			t.Passed, t.Failed, t.Errored, t.Skipped)
		fmt.Fprintf(w, "  Failure rate: %.2f%%\n", t.FailureRate*100)
		fmt.Fprintf(w, "  Duration ms: min=%d avg=%.1f max=%d\n",
			t.DurationMinMS, t.DurationAvgMS, t.DurationMaxMS)
		if len(t.Evidence) > 0 {
			fmt.Fprintln(w, "  Observed evidence:")
			for _, e := range t.Evidence {
				fmt.Fprintf(w, "    - %s\n", e)
			}
		}
		fmt.Fprintln(w, "  (Evidence only — not a proven root cause.)")
	}
	fmt.Fprintln(w)
}
