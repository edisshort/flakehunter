package analysis

import (
	"testing"

	"github.com/edisshort/flakehunter/internal/model"
)

func TestCompareHighlightsDelta(t *testing.T) {
	leftExp := model.Experiment{ID: "L", Parallel: 1, OrderMode: model.OrderDefault, RunsRequested: 5}
	rightExp := model.Experiment{ID: "R", Parallel: 4, OrderMode: model.OrderDefault, RunsRequested: 5}

	left := repeatOutcomes("a::t", 5, model.TestPass)
	right := append(repeatOutcomes("a::t", 2, model.TestPass), repeatOutcomes("a::t", 3, model.TestFail)...)

	rep := Compare(leftExp, rightExp, left, right, Thresholds{MinOutcomes: 1})
	if len(rep.Tests) != 1 {
		t.Fatalf("%+v", rep.Tests)
	}
	if rep.Tests[0].Delta <= 0 {
		t.Fatalf("expected positive delta, got %+v", rep.Tests[0])
	}
}

func TestPredecessorEvidence(t *testing.T) {
	execs := []model.Execution{
		{
			TestOrder: []string{"setup::a", "target::t"},
			Tests: []model.TestCase{
				{Name: "setup::a", Status: model.TestPass},
				{Name: "target::t", Status: model.TestFail},
			},
		},
		{
			TestOrder: []string{"setup::a", "target::t"},
			Tests: []model.TestCase{
				{Name: "setup::a", Status: model.TestPass},
				{Name: "target::t", Status: model.TestFail},
			},
		},
		{
			TestOrder: []string{"other::b", "target::t"},
			Tests: []model.TestCase{
				{Name: "other::b", Status: model.TestPass},
				{Name: "target::t", Status: model.TestPass},
			},
		},
	}
	rep := AnalyzeExperiment(model.Experiment{ID: "e"}, execs, Thresholds{MinOutcomes: 1})
	var target model.TestStats
	for _, ts := range rep.Tests {
		if ts.Name == "target::t" {
			target = ts
		}
	}
	if len(target.TopPredecessors) == 0 || target.TopPredecessors[0].Name != "setup::a" {
		t.Fatalf("%+v", target)
	}
}
