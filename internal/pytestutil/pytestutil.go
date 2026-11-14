// Package pytestutil prepares pytest commands so each run writes its own JUnit XML.
package pytestutil

import (
	"path/filepath"
	"strings"
)

// LooksLikePytest reports whether the argv likely invokes pytest.
func LooksLikePytest(argv []string) bool {
	for i, a := range argv {
		base := filepath.Base(a)
		if base == "pytest" || base == "pytest.exe" {
			return true
		}
		if a == "-m" && i+1 < len(argv) && argv[i+1] == "pytest" {
			return true
		}
	}
	return false
}

// WithJUnitXML returns a copy of argv that writes JUnit output to junitPath.
// If the command already has --junitxml / --junit-xml, that value is replaced.
func WithJUnitXML(argv []string, junitPath string) []string {
	out := make([]string, 0, len(argv)+2)
	skipNext := false
	replaced := false
	for i := 0; i < len(argv); i++ {
		if skipNext {
			skipNext = false
			continue
		}
		a := argv[i]
		if a == "--junitxml" || a == "--junit-xml" {
			out = append(out, a, junitPath)
			skipNext = true
			replaced = true
			continue
		}
		if strings.HasPrefix(a, "--junitxml=") || strings.HasPrefix(a, "--junit-xml=") {
			key := "--junitxml="
			if strings.HasPrefix(a, "--junit-xml=") {
				key = "--junit-xml="
			}
			out = append(out, key+junitPath)
			replaced = true
			continue
		}
		out = append(out, a)
	}
	if !replaced {
		out = append(out, "--junitxml="+junitPath)
	}
	return out
}
