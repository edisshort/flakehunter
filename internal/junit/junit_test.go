package junit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edisshort/flakehunter/internal/model"
)

func TestParsePassedFailedSkipped(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<testsuites>
  <testsuite name="pytest" tests="3">
    <testcase classname="tests.test_a" name="test_ok" time="0.010"/>
    <testcase classname="tests.test_a" name="test_bad" time="0.020">
      <failure message="assert 1 == 2">body</failure>
    </testcase>
    <testcase classname="tests.test_a" name="test_skip" time="0.001">
      <skipped message="not ready"/>
    </testcase>
  </testsuite>
</testsuites>`

	cases, err := Parse([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("got %d cases", len(cases))
	}
	if cases[0].Status != model.TestPass || cases[0].Name != "tests.test_a::test_ok" {
		t.Fatalf("case0=%+v", cases[0])
	}
	if cases[1].Status != model.TestFail || cases[1].Message != "assert 1 == 2" {
		t.Fatalf("case1=%+v", cases[1])
	}
	if cases[2].Status != model.TestSkip {
		t.Fatalf("case2=%+v", cases[2])
	}
}

func TestParseSingleTestsuite(t *testing.T) {
	xml := `<testsuite name="s" tests="0"></testsuite>`
	cases, err := Parse([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 0 {
		t.Fatalf("expected 0, got %d", len(cases))
	}
}

func TestParseMalformed(t *testing.T) {
	_, err := Parse([]byte("<not-junit>"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "nope.xml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFileOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "junit.xml")
	content := `<testsuite><testcase name="t" classname="c" time="1.5"/></testsuite>`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].DurationMS != 1500 {
		t.Fatalf("%+v", cases)
	}
}
