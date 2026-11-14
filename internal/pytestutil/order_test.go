package pytestutil

import (
	"strings"
	"testing"
)

func TestWithRandomlySeedAppend(t *testing.T) {
	out := WithRandomlySeed([]string{"python", "-m", "pytest"}, 42)
	joined := strings.Join(out, " ")
	if !strings.Contains(joined, "--randomly-seed=42") {
		t.Fatalf("%v", out)
	}
}

func TestWithRandomlySeedReplace(t *testing.T) {
	out := WithRandomlySeed([]string{"pytest", "--randomly-seed", "1"}, 99)
	found := false
	for i, a := range out {
		if a == "--randomly-seed" && i+1 < len(out) && out[i+1] == "99" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%v", out)
	}
}

func TestEnvWithPythonHashSeed(t *testing.T) {
	env := EnvWithPythonHashSeed(7)
	ok := false
	for _, e := range env {
		if e == "PYTHONHASHSEED=7" {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("%v", env)
	}
}
