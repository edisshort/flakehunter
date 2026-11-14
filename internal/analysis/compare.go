package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/edisshort/flakehunter/internal/model"
)

// Compare contrasts failure rates for tests that appear in either experiment.
// Useful for sequential vs parallel, or default vs random-order trials.
func Compare(leftExp, rightExp model.Experiment, leftExecs, rightExecs []model.Execution, th Thresholds) model.CompareReport {
	left := AnalyzeExperiment(leftExp, leftExecs, th)
	right := AnalyzeExperiment(rightExp, rightExecs, th)

	leftBy := map[string]model.TestStats{}
	for _, t := range left.Tests {
		leftBy[t.Name] = t
	}
	rightBy := map[string]model.TestStats{}
	for _, t := range right.Tests {
		rightBy[t.Name] = t
	}

	names := map[string]bool{}
	for n := range leftBy {
		names[n] = true
	}
	for n := range rightBy {
		names[n] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	rep := model.CompareReport{
		LeftID:  leftExp.ID,
		RightID: rightExp.ID,
		LeftMeta: model.CompareMeta{
			Parallel:  leftExp.Parallel,
			OrderMode: leftExp.OrderMode,
			Seed:      leftExp.Seed,
			Runs:      leftExp.RunsRequested,
		},
		RightMeta: model.CompareMeta{
			Parallel:  rightExp.Parallel,
			OrderMode: rightExp.OrderMode,
			Seed:      rightExp.Seed,
			Runs:      rightExp.RunsRequested,
		},
		Notes: []string{
			"Delta = right failure rate − left failure rate among decisive outcomes.",
			"A higher rate under parallel/random order is association evidence, not proof of root cause.",
		},
	}

	for _, name := range sorted {
		l, lok := leftBy[name]
		r, rok := rightBy[name]
		row := model.CompareTestRow{Name: name}
		if lok {
			row.LeftFailureRate = l.FailureRate
			row.LeftDecisive = l.Decisive
		}
		if rok {
			row.RightFailureRate = r.FailureRate
			row.RightDecisive = r.Decisive
		}
		row.Delta = round4(row.RightFailureRate - row.LeftFailureRate)
		row.Evidence = compareEvidence(row, leftExp, rightExp)
		rep.Tests = append(rep.Tests, row)
	}

	// Surface interesting rows first: largest absolute delta, then name.
	sort.SliceStable(rep.Tests, func(i, j int) bool {
		ai := math.Abs(rep.Tests[i].Delta)
		aj := math.Abs(rep.Tests[j].Delta)
		if ai == aj {
			return rep.Tests[i].Name < rep.Tests[j].Name
		}
		return ai > aj
	})
	return rep
}

func compareEvidence(row model.CompareTestRow, left, right model.Experiment) string {
	if row.LeftDecisive == 0 && row.RightDecisive == 0 {
		return "No decisive outcomes on either side."
	}
	if math.Abs(row.Delta) < 0.0001 {
		return "Failure rates are similar across configurations."
	}
	side := "right"
	if row.Delta < 0 {
		side = "left"
	}
	return fmt.Sprintf(
		"Higher failure association on %s (parallel %d/%d, order %s/%s). Evidence only.",
		side, left.Parallel, right.Parallel, emptyDefault(left.OrderMode), emptyDefault(right.OrderMode),
	)
}

func emptyDefault(s string) string {
	if s == "" {
		return model.OrderDefault
	}
	return s
}
