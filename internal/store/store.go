// Package store persists experiments as JSON files under .flakehunter/.
// No database in the MVP — files are easy to inspect and gitignore.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/edisshort/flakehunter/internal/model"
)

const (
	RootDirName       = ".flakehunter"
	ExperimentsDir    = "experiments"
	ExecutionsDirName = "executions"
)

// Store writes and reads experiment data relative to a root working directory.
type Store struct {
	// Root is usually the user's project directory (contains .flakehunter/).
	Root string
}

func New(root string) *Store {
	if root == "" {
		root = "."
	}
	return &Store{Root: root}
}

func (s *Store) BaseDir() string {
	return filepath.Join(s.Root, RootDirName)
}

func (s *Store) ExperimentDir(id string) string {
	return filepath.Join(s.BaseDir(), ExperimentsDir, id)
}

func (s *Store) ExecutionDir(experimentID string, runIndex int) string {
	return filepath.Join(s.ExperimentDir(experimentID), ExecutionsDirName, fmt.Sprintf("%04d", runIndex))
}

// NewExperimentID returns a readable unique id: 20060102-150405-<suffix>
func NewExperimentID(now time.Time) string {
	return now.UTC().Format("20060102-150405") + fmt.Sprintf("-%04d", now.Nanosecond()%10000)
}

// CreateExperiment creates folders and writes meta.json.
func (s *Store) CreateExperiment(exp model.Experiment) error {
	dir := s.ExperimentDir(exp.ID)
	if err := os.MkdirAll(filepath.Join(dir, ExecutionsDirName), 0o755); err != nil {
		return err
	}
	return s.WriteExperiment(exp)
}

func (s *Store) WriteExperiment(exp model.Experiment) error {
	return writeJSON(filepath.Join(s.ExperimentDir(exp.ID), "meta.json"), exp)
}

func (s *Store) ReadExperiment(id string) (model.Experiment, error) {
	var exp model.Experiment
	err := readJSON(filepath.Join(s.ExperimentDir(id), "meta.json"), &exp)
	return exp, err
}

// PrepareExecutionDir creates the per-run folder and returns absolute paths
// for stdout, stderr, junit, and result.json.
// Absolute paths matter: when the test command uses a different working
// directory, relative --junitxml paths would resolve in the wrong place.
func (s *Store) PrepareExecutionDir(experimentID string, runIndex int) (execDir, stdout, stderr, junit, result string, err error) {
	execDir = s.ExecutionDir(experimentID, runIndex)
	if err = os.MkdirAll(execDir, 0o755); err != nil {
		return
	}
	execDir, err = filepath.Abs(execDir)
	if err != nil {
		return
	}
	stdout = filepath.Join(execDir, "stdout.log")
	stderr = filepath.Join(execDir, "stderr.log")
	junit = filepath.Join(execDir, "junit.xml")
	result = filepath.Join(execDir, "result.json")
	return
}

func (s *Store) WriteExecution(experimentID string, runIndex int, exec model.Execution) error {
	path := filepath.Join(s.ExecutionDir(experimentID, runIndex), "result.json")
	return writeJSON(path, exec)
}

func (s *Store) ReadExecution(experimentID string, runIndex int) (model.Execution, error) {
	var exec model.Execution
	path := filepath.Join(s.ExecutionDir(experimentID, runIndex), "result.json")
	err := readJSON(path, &exec)
	return exec, err
}

// LoadExecutions reads result.json for run indexes 1..runsRequested.
// Missing files are skipped (partial experiments).
func (s *Store) LoadExecutions(experimentID string, runsRequested int) ([]model.Execution, error) {
	var out []model.Execution
	for i := 1; i <= runsRequested; i++ {
		exec, err := s.ReadExecution(experimentID, i)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return out, err
		}
		out = append(out, exec)
	}
	return out, nil
}

func (s *Store) WriteAnalysis(experimentID string, report model.AnalysisReport) error {
	return writeJSON(filepath.Join(s.ExperimentDir(experimentID), "analysis.json"), report)
}

func (s *Store) ReadAnalysis(experimentID string) (model.AnalysisReport, error) {
	var rep model.AnalysisReport
	err := readJSON(filepath.Join(s.ExperimentDir(experimentID), "analysis.json"), &rep)
	return rep, err
}

func (s *Store) WriteFullReport(experimentID string, rep model.FullReport) error {
	return writeJSON(filepath.Join(s.ExperimentDir(experimentID), "report.json"), rep)
}

func (s *Store) ReadFullReport(experimentID string) (model.FullReport, error) {
	var rep model.FullReport
	err := readJSON(filepath.Join(s.ExperimentDir(experimentID), "report.json"), &rep)
	return rep, err
}

func (s *Store) WriteInspection(experimentID, safeName string, insp model.TestInspection) error {
	dir := filepath.Join(s.ExperimentDir(experimentID), "inspections")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, safeName+".json"), insp)
}

func (s *Store) ListExperimentIDs() ([]string, error) {
	dir := filepath.Join(s.BaseDir(), ExperimentsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// LatestExperimentID returns the newest id by name (IDs are time-prefixed).
func (s *Store) LatestExperimentID() (string, error) {
	ids, err := s.ListExperimentIDs()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no experiments found")
	}
	latest := ids[0]
	for _, id := range ids[1:] {
		if id > latest {
			latest = id
		}
	}
	return latest, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
