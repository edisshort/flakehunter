// Package report formats analysis results for the terminal and JSON files.
package report

import (
	"fmt"
	"io"
	"strings"
	"time"

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
	fmt.Fprintf(w, "Min decisive outcomes:   %d\n", rep.MinOutcomes)
	if rep.OrderMode != "" {
		fmt.Fprintf(w, "Order mode:              %s\n", rep.OrderMode)
	}
	if rep.Seed != 0 {
		fmt.Fprintf(w, "Base seed:               %d\n", rep.Seed)
	}
	if rep.Parallel > 0 {
		fmt.Fprintf(w, "Parallel:                %d\n", rep.Parallel)
	}
	fmt.Fprintln(w)

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

	if rep.Variation != nil && len(rep.Variation.OrderLimitations) > 0 {
		fmt.Fprintln(w, "Order / variation limitations:")
		for _, lim := range rep.Variation.OrderLimitations {
			fmt.Fprintf(w, "  - %s\n", lim)
		}
		fmt.Fprintln(w)
	}

	printSection(w, "Flaky tests", rep, model.ClassFlaky)
	printSection(w, "Consistently failing tests", rep, model.ClassConsistentlyFailing)
	printSection(w, "Inconsistent presence", rep, model.ClassInconsistentPresence)
}

// WriteFullTerminal prints analysis plus execution history.
func WriteFullTerminal(w io.Writer, full model.FullReport) {
	WriteTerminal(w, full.Analysis)
	WriteExecutionHistory(w, full)
	fmt.Fprintf(w, "Experiment command: %v\n", full.Experiment.Command)
	fmt.Fprintf(w, "Generated at (UTC):  %s\n", full.GeneratedAt.Format(time.RFC3339))
}

// WriteCompare prints a configuration comparison report.
func WriteCompare(w io.Writer, rep model.CompareReport) {
	fmt.Fprintln(w, "FlakeHunter Compare Report")
	fmt.Fprintln(w, strings.Repeat("=", 28))
	fmt.Fprintf(w, "Left:  %s  parallel=%d order=%s seed=%d runs=%d\n",
		rep.LeftID, rep.LeftMeta.Parallel, empty(rep.LeftMeta.OrderMode), rep.LeftMeta.Seed, rep.LeftMeta.Runs)
	fmt.Fprintf(w, "Right: %s  parallel=%d order=%s seed=%d runs=%d\n\n",
		rep.RightID, rep.RightMeta.Parallel, empty(rep.RightMeta.OrderMode), rep.RightMeta.Seed, rep.RightMeta.Runs)

	for _, note := range rep.Notes {
		fmt.Fprintf(w, "Note: %s\n", note)
	}
	if len(rep.Notes) > 0 {
		fmt.Fprintln(w)
	}

	shown := 0
	for _, row := range rep.Tests {
		if row.LeftDecisive == 0 && row.RightDecisive == 0 {
			continue
		}
		// Show rows with any decisive data; highlight non-zero deltas first (already sorted).
		fmt.Fprintf(w, "%s\n", row.Name)
		fmt.Fprintf(w, "  left rate=%.2f%% (%d decisive)  right rate=%.2f%% (%d decisive)  delta=%+.2f%%\n",
			row.LeftFailureRate*100, row.LeftDecisive, row.RightFailureRate*100, row.RightDecisive, row.Delta*100)
		if row.Evidence != "" {
			fmt.Fprintf(w, "  %s\n", row.Evidence)
		}
		fmt.Fprintln(w)
		shown++
		if shown >= 20 {
			fmt.Fprintln(w, "(showing top 20 rows by |delta|)")
			break
		}
	}
	if shown == 0 {
		fmt.Fprintln(w, "No comparable per-test results.")
	}
}

func empty(s string) string {
	if s == "" {
		return model.OrderDefault
	}
	return s
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
		if len(t.TopPredecessors) > 0 {
			fmt.Fprintln(w, "  Top predecessors before failure:")
			for _, p := range t.TopPredecessors {
				fmt.Fprintf(w, "    - %s (%d)\n", p.Name, p.Count)
			}
		}
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
