# FlakeHunter

CLI tool that repeatedly runs automated tests, stores per-run results, and detects flaky tests with deterministic statistics (no LLM APIs).

**Status:** Phase 5 — analysis + bounded parallel execution.

## Requirements

- Go 1.22+ (developed with Go 1.27)
- For pytest experiments: Python + pytest

## Quick start

```powershell
go test ./...
go test -race ./...
go build -o flakehunter.exe ./cmd/flakehunter

.\flakehunter.exe run --runs 5 -- go version
.\flakehunter.exe run --runs 8 --parallel 4 -- go version
.\flakehunter.exe run --runs 10 --timeout 2m -- python -m pytest
.\flakehunter.exe report
.\flakehunter.exe list
```

Results live under `.flakehunter/experiments/<id>/` (`meta.json`, `executions/`, `analysis.json`).

## Classification (Phase 4)

For each test (from JUnit/pytest results):

| Class | Meaning |
|------|---------|
| `stable_passing` | Enough decisive runs, never failed |
| `consistently_failing` | Enough decisive runs, never passed |
| `flaky` | Enough decisive runs, mixed pass and fail/error |
| `insufficient_evidence` | Fewer decisive outcomes than `--min-outcomes` (default 5) |
| `inconsistent_presence` | Missing from some runs that produced results |

Skips do not count as pass or fail. Reports show **evidence**, not proven root causes.

## Parallelism (Phase 5)

`--parallel N` starts at most N workers. Each run writes to its own folder (`executions/0001`, `0002`, …) so workers do not share result files. Logging uses a mutex. Cancel with Ctrl+C; in-flight processes are stopped via context.

Parallel failures are a signal to investigate — not automatic proof of a concurrency bug in your code.

## Phases

| Phase | Focus | Status |
|------|--------|--------|
| 1 | CLI foundation | done |
| 2 | Subprocess engine | done |
| 3 | pytest + JUnit XML | done |
| 4 | Flakiness analysis | done |
| 5 | Parallel worker pool | done |
| 6 | Order / seed variation | next |
| 7+ | Richer reports, validation, polish | planned |

## License

MIT (to be confirmed)
