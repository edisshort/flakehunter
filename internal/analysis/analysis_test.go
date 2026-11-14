package analysis

import (
	"testing"

	"github.com/edisshort/flakehunter/internal/model"
)

func TestClassifyFlaky(t *testing.T) {
	execs := repeatOutcomes("a::t", 5, model.TestPass)
	execs = append(execs, outcome("a::t", model.TestFail)...)
	// pad to 6 decisive with min=5
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	if rep.Counts.Flaky != 1 {
		t.Fatalf("counts=%+v tests=%+v", rep.Counts, rep.Tests)
	}
	if rep.Tests[0].Classification != model.ClassFlaky {
		t.Fatalf("%+v", rep.Tests[0])
	}
}

func TestClassifyStable(t *testing.T) {
	execs := repeatOutcomes("a::t", 5, model.TestPass)
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	if rep.Counts.StablePassing != 1 {
		t.Fatalf("%+v", rep.Counts)
	}
}

func TestClassifyConsistentFail(t *testing.T) {
	execs := repeatOutcomes("a::t", 5, model.TestFail)
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	if rep.Counts.ConsistentlyFailing != 1 {
		t.Fatalf("%+v", rep.Counts)
	}
}

func TestInsufficientEvidence(t *testing.T) {
	execs := repeatOutcomes("a::t", 2, model.TestPass)
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	if rep.Counts.InsufficientEvidence != 1 {
		t.Fatalf("%+v", rep.Counts)
	}
}

func TestSkipsDoNotCountAsDecisiveFail(t *testing.T) {
	// 5 passes + many skips still stable
	execs := repeatOutcomes("a::t", 5, model.TestPass)
	execs = append(execs, outcome("a::t", model.TestSkip)...)
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	if rep.Tests[0].Classification != model.ClassStablePassing {
		t.Fatalf("%+v", rep.Tests[0])
	}
	if rep.Tests[0].Decisive != 5 {
		t.Fatalf("decisive=%d", rep.Tests[0].Decisive)
	}
}

func TestInconsistentPresence(t *testing.T) {
	execs := []model.Execution{
		{RunIndex: 1, Tests: []model.TestCase{{Name: "only::here", Status: model.TestPass}}},
		{RunIndex: 2, Tests: []model.TestCase{{Name: "other::t", Status: model.TestPass}}},
		{RunIndex: 3, Tests: []model.TestCase{{Name: "other::t", Status: model.TestPass}}},
	}
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 5})
	var found bool
	for _, ts := range rep.Tests {
		if ts.Name == "only::here" {
			found = true
			if ts.Classification != model.ClassInconsistentPresence {
				t.Fatalf("%+v", ts)
			}
		}
	}
	if !found {
		t.Fatal("missing test")
	}
}

func TestFailureRate(t *testing.T) {
	execs := append(repeatOutcomes("a::t", 3, model.TestPass), repeatOutcomes("a::t", 1, model.TestFail)...)
	rep := Analyze("e1", execs, Thresholds{MinOutcomes: 1})
	if rep.Tests[0].FailureRate != 0.25 {
		t.Fatalf("rate=%v", rep.Tests[0].FailureRate)
	}
}

func TestNoTestsNote(t *testing.T) {
	rep := Analyze("e1", []model.Execution{{RunIndex: 1}}, DefaultThresholds())
	if len(rep.Notes) == 0 {
		t.Fatal("expected note")
	}
}

func outcome(name, status string) []model.Execution {
	return []model.Execution{{
		Tests: []model.TestCase{{Name: name, Status: status, DurationMS: 10}},
	}}
}

func repeatOutcomes(name string, n int, status string) []model.Execution {
	var out []model.Execution
	for i := 0; i < n; i++ {
		out = append(out, outcome(name, status)...)
	}
	return out
}
