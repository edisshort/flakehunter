// Package junit parses JUnit XML reports produced by pytest (--junitxml=...).
package junit

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/edisshort/flakehunter/internal/model"
)

// XML structures mirror a common subset of JUnit / xUnit schemas.
// pytest writes <testsuites> or a single <testsuite>.

type testsuites struct {
	XMLName xml.Name    `xml:"testsuites"`
	Suites  []testsuite `xml:"testsuite"`
}

type testsuite struct {
	XMLName xml.Name   `xml:"testsuite"`
	Name    string     `xml:"name,attr"`
	Cases   []testcase `xml:"testcase"`
}

type testcase struct {
	Name      string     `xml:"name,attr"`
	Classname string     `xml:"classname,attr"`
	Time      float64    `xml:"time,attr"` // seconds
	Failure   *failure   `xml:"failure"`
	Error     *failure   `xml:"error"`
	Skipped   *skipped   `xml:"skipped"`
}

type failure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type skipped struct {
	Message string `xml:"message,attr"`
}

// ParseFile reads a JUnit XML file and returns individual test cases.
func ParseFile(path string) ([]model.TestCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse converts JUnit XML bytes into model.TestCase values.
func Parse(data []byte) ([]model.TestCase, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("empty junit xml")
	}

	// Prefer <testsuites> wrapper; fall back to a lone <testsuite>.
	var suites testsuites
	if err := xml.Unmarshal(data, &suites); err == nil && (len(suites.Suites) > 0 || strings.Contains(trimmed, "<testsuites")) {
		return fromSuites(suites.Suites), nil
	}

	var suite testsuite
	if err := xml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parse junit xml: %w", err)
	}
	return fromSuites([]testsuite{suite}), nil
}

func fromSuites(suites []testsuite) []model.TestCase {
	var out []model.TestCase
	for _, s := range suites {
		for _, c := range s.Cases {
			out = append(out, toModel(c))
		}
	}
	return out
}

func toModel(c testcase) model.TestCase {
	name := c.Name
	if c.Classname != "" {
		// pytest classname is often a dotted module path; combine for identity.
		name = c.Classname + "::" + c.Name
	}
	tc := model.TestCase{
		Name:       name,
		Classname:  c.Classname,
		DurationMS: int64(c.Time * 1000),
		Status:     model.TestPass,
	}
	switch {
	case c.Failure != nil:
		tc.Status = model.TestFail
		tc.Message = messageOf(c.Failure)
	case c.Error != nil:
		tc.Status = model.TestError
		tc.Message = messageOf(c.Error)
	case c.Skipped != nil:
		tc.Status = model.TestSkip
		tc.Message = c.Skipped.Message
	}
	return tc
}

func messageOf(f *failure) string {
	if f.Message != "" {
		return strings.TrimSpace(f.Message)
	}
	return strings.TrimSpace(f.Body)
}
