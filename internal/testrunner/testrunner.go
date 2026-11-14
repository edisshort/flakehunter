// Package testrunner detects common test CLIs and prepares JUnit output flags.
package testrunner

import (
	"path/filepath"
	"strings"

	"github.com/edisshort/flakehunter/internal/pytestutil"
)

// Kind identifies which integration FlakeHunter should use.
type Kind int

const (
	KindNone Kind = iota
	KindPytest
	KindVitest
)

// Detect inspects argv and returns the best-matching runner kind.
func Detect(argv []string) Kind {
	if pytestutil.LooksLikePytest(argv) {
		return KindPytest
	}
	if LooksLikeVitest(argv) {
		return KindVitest
	}
	return KindNone
}

// LooksLikeVitest reports vitest or npm/yarn/pnpm test scripts that typically run vitest.
func LooksLikeVitest(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	for _, a := range argv {
		base := strings.ToLower(filepath.Base(a))
		base = strings.TrimSuffix(base, ".cmd")
		base = strings.TrimSuffix(base, ".exe")
		if base == "vitest" {
			return true
		}
	}
	// npm test / yarn test / pnpm test
	if isNodePackageManager(argv[0]) && len(argv) >= 2 && argv[1] == "test" {
		return true
	}
	joined := strings.ToLower(strings.Join(argv, " "))
	return strings.Contains(joined, "vitest")
}

func isNodePackageManager(bin string) bool {
	base := strings.ToLower(filepath.Base(bin))
	base = strings.TrimSuffix(base, ".cmd")
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "npm", "npx", "yarn", "pnpm", "bun":
		return true
	default:
		return false
	}
}

// WithJUnitXML injects runner-specific flags so results land at junitPath.
func WithJUnitXML(kind Kind, argv []string, junitPath string) []string {
	switch kind {
	case KindPytest:
		return pytestutil.WithJUnitXML(argv, junitPath)
	case KindVitest:
		return withVitestJUnit(argv, junitPath)
	default:
		return append([]string{}, argv...)
	}
}

func withVitestJUnit(argv []string, junitPath string) []string {
	out := append([]string{}, argv...)

	// npm/yarn/pnpm test needs "--" before vitest flags.
	if len(out) >= 2 && isNodePackageManager(out[0]) && out[1] == "test" {
		dash := -1
		for i, a := range out {
			if a == "--" {
				dash = i
				break
			}
		}
		if dash < 0 {
			out = append(out, "--")
			dash = len(out) - 1
		}
		head := append([]string{}, out[:dash+1]...)
		tail := stripVitestJUnitFlags(out[dash+1:])
		tail = append(tail, "--reporter=junit", "--outputFile="+junitPath)
		return append(head, tail...)
	}

	// Direct vitest / npx vitest invocation.
	tail := stripVitestJUnitFlags(out)
	return append(tail, "--reporter=junit", "--outputFile="+junitPath)
}

func stripVitestJUnitFlags(args []string) []string {
	out := make([]string, 0, len(args))
	skipNext := false
	for i := 0; i < len(args); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		a := args[i]
		if a == "--outputFile" || a == "--output-file" {
			skipNext = true
			continue
		}
		if strings.HasPrefix(a, "--outputFile=") || strings.HasPrefix(a, "--output-file=") {
			continue
		}
		if a == "--reporter" && i+1 < len(args) && (args[i+1] == "junit" || strings.HasPrefix(args[i+1], "junit")) {
			skipNext = true
			continue
		}
		if a == "--reporter=junit" || strings.HasPrefix(a, "--reporter=junit") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// SupportsRandomOrder is true when FlakeHunter can inject a reproducible seed.
func SupportsRandomOrder(kind Kind) bool {
	return kind == KindPytest
}
