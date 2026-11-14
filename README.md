# FlakeHunter

CLI tool that repeatedly runs automated tests, stores per-run results, and detects flaky tests with deterministic statistics (no LLM APIs).

**Status:** Phase 8 — reporting + validated intentional flaky suite.

## Requirements

- Go 1.22+ (developed with Go 1.27)
- For pytest experiments: Python + pytest
- For `--order random`: [pytest-randomly](https://pypi.org/project/pytest-randomly/)

## Quick start

```powershell
go test ./...
go test -race ./...
go build -o flakehunter.exe ./cmd/flakehunter

.\flakehunter.exe help
.\flakehunter.exe run --runs 20 --min-outcomes 5 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe report --workdir testdata/flaky_suite
.\flakehunter.exe inspect test_coin_flip --workdir testdata/flaky_suite
.\flakehunter.exe list --workdir testdata/flaky_suite
```

Artifacts: `.flakehunter/experiments/<id>/{meta,analysis,report}.json`, `executions/`, `inspections/`.

## Commands

| Command | Purpose |
|---------|---------|
| `run` | Repeat a test command (sequential or `--parallel N`) |
| `report` | Terminal summary + `analysis.json` + `report.json` |
| `inspect` | Per-test history and evidence |
| `compare` | Contrast two experiments |
| `list` | List saved experiments |

## Classification

| Class | Meaning |
|------|---------|
| `stable_passing` | Enough decisive runs, never failed |
| `consistently_failing` | Enough decisive runs, never passed |
| `flaky` | Mixed pass and fail/error |
| `insufficient_evidence` | Below `--min-outcomes` (default 5) |
| `inconsistent_presence` | Missing from some collected runs |

## Phase 8 demo suite

See [`testdata/flaky_suite`](testdata/flaky_suite) and measured results in [`docs/VALIDATION.md`](docs/VALIDATION.md).

Real measured highlight (this machine, 2026-11-14):

- `test_coin_flip` → **flaky** (30% fail over 20 sequential runs)
- `test_always_fails` → **consistently_failing** (20/20)
- `test_shared_counter_increment` → stable sequentially, **flaky at 58%** under `--parallel 4`

## Honest limitations

- JUnit order is best-effort.
- Random order needs a pytest plugin.
- Compare/parallel signals are association evidence, not root-cause proof.
- Do not invent performance claims; re-measure on your machine.

## Phases

| Phase | Focus | Status |
|------|--------|--------|
| 1–6 | CLI → engine → JUnit → analysis → parallel → order/compare | done |
| 7 | Rich reports + `inspect` | done |
| 8 | Intentional flaky suite + validation doc | done |
| 9 | GitHub polish / CI | next |

## License

MIT (to be confirmed)
