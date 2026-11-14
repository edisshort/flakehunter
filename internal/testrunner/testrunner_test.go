package testrunner

import (
	"strings"
	"testing"
)

func TestDetectPytest(t *testing.T) {
	if Detect([]string{"python", "-m", "pytest"}) != KindPytest {
		t.Fatal("expected pytest")
	}
}

func TestDetectVitestNPM(t *testing.T) {
	if Detect([]string{"npm", "test"}) != KindVitest {
		t.Fatal("expected vitest via npm test")
	}
}

func TestDetectVitestDirect(t *testing.T) {
	if Detect([]string{"npx", "vitest", "run"}) != KindVitest {
		t.Fatal("expected vitest")
	}
}

func TestDetectNone(t *testing.T) {
	if Detect([]string{"go", "test", "./..."}) != KindNone {
		t.Fatal("expected none")
	}
}

func TestWithVitestJUnitNPM(t *testing.T) {
	out := WithJUnitXML(KindVitest, []string{"npm", "test"}, `C:\tmp\junit.xml`)
	joined := strings.Join(out, " ")
	if !strings.Contains(joined, "--") {
		t.Fatalf("missing -- : %v", out)
	}
	if !strings.Contains(joined, "--reporter=junit") {
		t.Fatalf("missing reporter: %v", out)
	}
	if !strings.Contains(joined, `--outputFile=C:\tmp\junit.xml`) {
		t.Fatalf("missing outputFile: %v", out)
	}
}

func TestWithVitestJUnitDirect(t *testing.T) {
	out := WithJUnitXML(KindVitest, []string{"vitest", "run"}, "/tmp/j.xml")
	joined := strings.Join(out, " ")
	if !strings.Contains(joined, "--reporter=junit") || !strings.Contains(joined, "--outputFile=/tmp/j.xml") {
		t.Fatalf("%v", out)
	}
}

func TestWithVitestReplacesOldFlags(t *testing.T) {
	out := WithJUnitXML(KindVitest, []string{"npm", "test", "--", "--reporter=junit", "--outputFile=old.xml"}, "new.xml")
	count := 0
	for _, a := range out {
		if strings.Contains(a, "old.xml") {
			t.Fatalf("old path remained: %v", out)
		}
		if a == "--reporter=junit" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("reporter count=%d out=%v", count, out)
	}
}
