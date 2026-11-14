# FlakeHunter

CLI tool that repeatedly runs automated tests, stores per-run results, and detects flaky tests with deterministic statistics (no LLM APIs).

**Status:** Phase 6 — order/seed variation + experiment compare.

## Requirements

- Go 1.22+ (developed with Go 1.27)
- For pytest experiments: Python + pytest
- For `--order random`: [pytest-randomly](https://pypi.org/project/pytest-randomly/) (provides `--randomly-seed`)

## Quick start

```powershell
go test ./...
go test -race ./...
go build -o flakehunter.exe ./cmd/flakehunter

.\flakehunter.exe run --runs 5 -- go version
.\flakehunter.exe run --runs 8 --parallel 4 -- go version
.\flakehunter.exe run --runs 10 --timeout 2m -- python -m pytest
.\flakehunter.exe run --runs 10 --order random --seed 42 -- python -m pytest
.\flakehunter.exe report
.\flakehunter.exe compare <exp-a> <exp-b>
.\flakehunter.exe list
```

Results live under `.flakehunter/experiments/<id>/`.

## Classification (Phase 4)

| Class | Meaning |
|------|---------|
| `stable_passing` | Enough decisive runs, never failed |
| `consistently_failing` | Enough decisive runs, never passed |
| `flaky` | Enough decisive runs, mixed pass and fail/error |
| `insufficient_evidence` | Fewer decisive outcomes than `--min-outcomes` (default 5) |
| `inconsistent_presence` | Missing from some runs that produced results |

## Parallelism (Phase 5)

`--parallel N` starts at most N workers. Each run writes to its own folder. Cancel with Ctrl+C.

## Order variation (Phase 6)

| Flag | Meaning |
|------|---------|
| `--order default` | Runner's normal order |
| `--order random` | Inject `--randomly-seed` (needs pytest-randomly) |
| `--seed N` | Base seed; run `i` uses `N+i` for reproducible variety |

Also records best-effort `test_order` from JUnit and predecessor counts before failures.

`flakehunter compare A B` contrasts failure rates across two configs (e.g. sequential vs parallel).

### Honest limitations

- JUnit document order may not equal true execution order.
- Random order depends on the test runner/plugin.
- Higher failure rate under parallel/random is **association evidence**, not proof of root cause.

## Phases

| Phase | Focus | Status |
|------|--------|--------|
| 1–5 | CLI, engine, JUnit, analysis, parallel | done |
| 6 | Order/seed + compare | done |
| 7+ | Richer reports, real-world validation, polish | next |

## License

MIT (to be confirmed)
