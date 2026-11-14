package experiment

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/edisshort/flakehunter/internal/model"
	"github.com/edisshort/flakehunter/internal/store"
)

func TestRunSequentialHelper(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	root := t.TempDir()
	helper, err := filepath.Abs(filepath.Join("..", "engine", "testdata", "helper", "main.go"))
	if err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	r := &Runner{
		Store:  store.New(root),
		Stdout: &out,
		Stderr: &errBuf,
	}
	exp, err := r.Run(context.Background(), Config{
		Command:  []string{"go", "run", helper, "ok"},
		Runs:     2,
		Parallel: 1,
		Timeout:  2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("err=%v stderr=%s", err, errBuf.String())
	}
	if exp.Status != model.StatusCompleted {
		t.Fatalf("status=%s out=%s", exp.Status, out.String())
	}
	for i := 1; i <= 2; i++ {
		resultPath := filepath.Join(store.New(root).ExecutionDir(exp.ID, i), "result.json")
		if _, err := os.Stat(resultPath); err != nil {
			t.Fatalf("missing %s: %v", resultPath, err)
		}
	}
}
