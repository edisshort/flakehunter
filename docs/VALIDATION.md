# Phase 8 validation — intentional flaky suite

This document records a **real** FlakeHunter run against `testdata/flaky_suite`.
Numbers below were produced on this machine; they are not invented benchmarks.

## Environment

| Item | Value |
|------|--------|
| Date (UTC) | 2026-11-14 |
| OS | Windows amd64 |
| Go | go1.27.4 |
| Python | 3.13.3 |
| pytest | 8.3.3 |
| FlakeHunter | 0.8.0-phase8 |
| Suite | `testdata/flaky_suite` |
| Command | `python -m pytest -q` |

## Experiment A — sequential

```text
flakehunter run --runs 20 --min-outcomes 5 --timeout 60s --workdir testdata/flaky_suite -- python -m pytest -q
```

- Experiment ID: `20261114-080011-0700`
- Parallel: 1
- Runs with test results: 20 / 20
- Unique tests: 7

| Classification | Count |
|----------------|------:|
| Stable passing | 4 |
| Flaky | 2 |
| Consistently failing | 1 |

### Notable tests (sequential)

| Test | Class | Pass / Fail | Failure rate |
|------|-------|-------------|--------------|
| `test_stable::test_always_passes` | stable_passing | 20 / 0 | 0% |
| `test_consistent_fail::test_always_fails` | consistently_failing | 0 / 20 | 100% |
| `test_coin_flip::test_coin_flip` | flaky | 14 / 6 | 30.00% |
| `test_timing::test_timing_window` | flaky | 12 / 8 | 40.00% |
| `test_shared_file::test_shared_counter_increment` | stable_passing | 20 / 0 | 0% |

Evidence note for timing: failed runs were slower on average (fail avg ~50ms vs pass avg ~23ms).

## Experiment B — parallel

```text
flakehunter run --runs 12 --parallel 4 --min-outcomes 5 --timeout 60s --workdir testdata/flaky_suite -- python -m pytest -q
```

- Experiment ID: `20261114-080138-1900`
- Parallel: 4
- Runs with test results: 12 / 12

| Classification | Count |
|----------------|------:|
| Stable passing | 3 |
| Flaky | 3 |
| Consistently failing | 1 |

### Notable change under parallel

| Test | Seq failure rate | Par failure rate | Delta |
|------|-----------------:|-----------------:|------:|
| `test_shared_file::test_shared_counter_increment` | 0% | 58.33% | +58.33% |
| `test_timing::test_timing_window` | 40% | 58.33% | +18.33% |
| `test_coin_flip::test_coin_flip` | 30% | 8.33% | −21.67% |
| `test_consistent_fail::test_always_fails` | 100% | 100% | 0 |

`compare` phrased the shared-file result as: higher failure association on the parallel side — **evidence, not proof** of a concurrency bug (though the fixture is intentionally racy).

Coin-flip variance between 20 vs 12 runs is expected for a random fixture; do not over-interpret the delta.

## Inspect

```text
flakehunter inspect test_coin_flip --experiment 20261114-080011-0700 --workdir testdata/flaky_suite
```

Confirmed mixed pass/fail history (14 pass / 6 fail) with per-run messages for failures.

## What worked

1. Per-test JUnit collection across 20 sequential and 12 parallel runs.
2. Correct separation of **consistently failing** vs **flaky**.
3. Shared-file fixture stayed stable sequentially and became flaky under `--parallel 4`.
4. `report.json` / `analysis.json` / `inspect` artifacts written under the suite’s `.flakehunter/`.

## Limitations / false positives

1. Predecessor stats for always-failing tests reflect collection order (e.g. always after coin_flip) — not a causal claim.
2. Order-dependent fixtures did not flake in these runs (flag file persisted on disk across processes).
3. Random fixtures have sampling noise; short parallel runs can under- or over-estimate rates.
4. No external open-source repo was run in this validation pass (suite is owned and intentional).
5. QSV was not targeted here; use the same CLI pattern when ready.

## Re-run

```powershell
go build -o flakehunter.exe ./cmd/flakehunter
.\flakehunter.exe run --runs 20 --min-outcomes 5 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe run --runs 12 --parallel 4 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe compare <seq-id> <par-id> --workdir testdata/flaky_suite
.\flakehunter.exe inspect test_coin_flip --workdir testdata/flaky_suite
```
