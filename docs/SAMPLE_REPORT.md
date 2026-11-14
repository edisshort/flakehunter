# Sample report (real run)

Source experiment: `20261114-080011-0700`  
Suite: `testdata/flaky_suite`  
Runs: 20 sequential, `python -m pytest -q`  
Full measured write-up: [VALIDATION.md](VALIDATION.md)

```text
FlakeHunter Experiment Report
================================
Experiment:              20261114-080011-0700
Executions analyzed:     20
Runs with test results:  20
Unique tests:            7
Min decisive outcomes:   5
Order mode:              default
Parallel:                1

Stable passing:          4
Flaky:                   2
Consistently failing:    1
Insufficient evidence:   0
Inconsistent presence:   0

Flaky tests (2)
-----------------

Name: test_coin_flip::test_coin_flip
  Appearances: 20 / 20
  Passed: 14  Failed: 6  Errored: 0  Skipped: 0
  Failure rate: 30.00%
  Observed evidence:
    - Appeared in 20 of 20 runs with collected test results.
    - Decisive outcomes: 20 (passed=14 failed=6 errored=0 skipped=0).
    - Failure rate among decisive outcomes: 30.00%.
  (Evidence only — not a proven root cause.)

Name: test_timing::test_timing_window
  Appearances: 20 / 20
  Passed: 12  Failed: 8  Errored: 0  Skipped: 0
  Failure rate: 40.00%
  Observed evidence:
    - Failed runs were slower on average (fail avg 50ms vs pass avg 23ms).
  (Evidence only — not a proven root cause.)

Consistently failing tests (1)
--------------------------------

Name: test_consistent_fail::test_always_fails
  Appearances: 20 / 20
  Passed: 0  Failed: 20
  Failure rate: 100.00%
```

Sanitized JSON excerpt: [samples/analysis_excerpt.json](samples/analysis_excerpt.json)
