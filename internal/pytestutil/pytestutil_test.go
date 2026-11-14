package pytestutil

import (
	"strings"
	"testing"
)

func TestLooksLikePytest(t *testing.T) {
	cases := []struct {
		argv []string
		want bool
	}{
		{[]string{"python", "-m", "pytest"}, true},
		{[]string{"pytest", "-q"}, true},
		{[]string{"python", "app.py"}, false},
		{[]string{"go", "test", "./..."}, false},
	}
	for _, tc := range cases {
		if got := LooksLikePytest(tc.argv); got != tc.want {
			t.Fatalf("argv=%v got=%v want=%v", tc.argv, got, tc.want)
		}
	}
}

func TestWithJUnitXMLAppend(t *testing.T) {
	out := WithJUnitXML([]string{"python", "-m", "pytest"}, `C:\tmp\junit.xml`)
	joined := strings.Join(out, " ")
	if !strings.Contains(joined, "--junitxml=") {
		t.Fatalf("missing junit flag: %v", out)
	}
}

func TestWithJUnitXMLReplace(t *testing.T) {
	out := WithJUnitXML([]string{"pytest", "--junitxml", "old.xml", "-q"}, "new.xml")
	found := 0
	for i, a := range out {
		if a == "--junitxml" && i+1 < len(out) && out[i+1] == "new.xml" {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("expected one replaced flag, got %v", out)
	}
}
