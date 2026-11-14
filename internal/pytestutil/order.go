package pytestutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// WithRandomlySeed injects/replaces --randomly-seed for pytest-randomly.
// If the plugin is not installed, pytest will error — FlakeHunter records that
// as an execution failure (honest limitation of random-order mode).
func WithRandomlySeed(argv []string, seed int64) []string {
	out := make([]string, 0, len(argv)+1)
	replaced := false
	skipNext := false
	for i := 0; i < len(argv); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		a := argv[i]
		if a == "--randomly-seed" {
			out = append(out, a, strconv.FormatInt(seed, 10))
			skipNext = true
			replaced = true
			continue
		}
		if strings.HasPrefix(a, "--randomly-seed=") {
			out = append(out, fmt.Sprintf("--randomly-seed=%d", seed))
			replaced = true
			continue
		}
		out = append(out, a)
	}
	if !replaced {
		out = append(out, fmt.Sprintf("--randomly-seed=%d", seed))
	}
	return out
}

// EnvWithPythonHashSeed returns a copy of the process environment with
// PYTHONHASHSEED set. This improves reproducibility of some Python hash-order
// effects; it is not a full substitute for test-order randomization plugins.
func EnvWithPythonHashSeed(seed int64) []string {
	env := os.Environ()
	key := "PYTHONHASHSEED="
	val := key + strconv.FormatInt(seed, 10)
	found := false
	for i, e := range env {
		if strings.HasPrefix(e, key) {
			env[i] = val
			found = true
			break
		}
	}
	if !found {
		env = append(env, val)
	}
	return env
}
