package report

import (
	"testing"

	"github.com/edisshort/flakehunter/internal/model"
)

func TestInspectExact(t *testing.T) {
	analysis := model.AnalysisReport{
		Tests: []model.TestStats{{
			Name:           "tests.test_a::test_flake",
			Classification: model.ClassFlaky,
			Passed:         2,
			Failed:         1,
			Decisive:       3,
		}},
	}
	execs := []model.Execution{
		{RunIndex: 1, Tests: []model.TestCase{{Name: "tests.test_a::test_flake", Status: model.TestPass}}},
		{RunIndex: 2, Tests: []model.TestCase{{Name: "tests.test_a::test_flake", Status: model.TestFail, Message: "boom"}}},
		{RunIndex: 3, Tests: []model.TestCase{{Name: "tests.test_a::test_flake", Status: model.TestPass}}},
	}
	insp := InspectTest("e1", analysis, execs, "tests.test_a::test_flake")
	if insp.Stats == nil || insp.Stats.Classification != model.ClassFlaky {
		t.Fatalf("%+v", insp)
	}
	if len(insp.History) != 3 || insp.History[1].Status != model.TestFail {
		t.Fatalf("%+v", insp.History)
	}
}

func TestInspectSubstringAmbiguous(t *testing.T) {
	analysis := model.AnalysisReport{
		Tests: []model.TestStats{
			{Name: "a::test_one"},
			{Name: "b::test_one"},
		},
	}
	insp := InspectTest("e1", analysis, nil, "test_one")
	if len(insp.Notes) < 2 {
		t.Fatalf("expected ambiguity notes: %+v", insp.Notes)
	}
}

func TestBuildFullReport(t *testing.T) {
	full := BuildFullReport(
		model.Experiment{ID: "e1", RunsRequested: 1},
		model.AnalysisReport{ExperimentID: "e1"},
		[]model.Execution{{RunIndex: 1, ExitCode: 0, Tests: []model.TestCase{{Name: "t", Status: model.TestPass}}}},
	)
	if len(full.Executions) != 1 || full.Executions[0].TestCount != 1 {
		t.Fatalf("%+v", full)
	}
}
