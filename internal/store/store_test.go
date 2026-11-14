package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/edisshort/flakehunter/internal/model"
)

func TestCreateAndReadExperiment(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	exp := model.Experiment{
		ID:            "test-exp",
		CreatedAt:     time.Now().UTC(),
		Command:       []string{"pytest"},
		RunsRequested: 3,
		Parallel:      1,
		Status:        model.StatusRunning,
	}
	if err := s.CreateExperiment(exp); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadExperiment("test-exp")
	if err != nil {
		t.Fatal(err)
	}
	if got.RunsRequested != 3 {
		t.Fatalf("%+v", got)
	}

	_, stdout, _, _, result, err := s.PrepareExecutionDir("test-exp", 1)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(stdout) != "stdout.log" {
		t.Fatalf("stdout path %s", stdout)
	}
	exec := model.Execution{ExperimentID: "test-exp", RunIndex: 1, ExitCode: 0}
	if err := s.WriteExecution("test-exp", 1, exec); err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("empty result path")
	}

	ids, err := s.ListExperimentIDs()
	if err != nil || len(ids) != 1 || ids[0] != "test-exp" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}
