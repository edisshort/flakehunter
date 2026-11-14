# FlakeHunter

**Detect flaky tests. Collect evidence. No AI guesswork.**

FlakeHunter is a Go CLI that repeatedly runs your automated tests, stores per-run results, and classifies which tests are stable, consistently failing, or flaky — using deterministic statistics only.

[Architecture](docs/ARCHITECTURE.md) · [Design decisions](docs/DESIGN.md) · [Validation](docs/VALIDATION.md) · [Sample report](docs/SAMPLE_REPORT.md) · [Roadmap](docs/ROADMAP.md)

---

## Why it exists

CI that is “red then green on retry” wastes hours. Exit codes alone do not explain *which* tests are unstable or *under what conditions*. FlakeHunter turns repeated runs into structured evidence.

## Features

- Repeat any test command N times (`--runs`)
- Sequential or bounded parallel workers (`--parallel`)
- pytest integration via JUnit XML (per-test pass/fail/skip)
- Classifications: stable / flaky / consistently failing / insufficient evidence
- Order/seed variation (`--order random`, needs pytest-randomly)
- `report`, `inspect`, and `compare` across experiments
- JSON artifacts under `.flakehunter/` (no database)
- **No LLM APIs**

## Install

Requirements: **Go 1.22+** (developed with Go 1.27).

```bash
git clone https://github.com/edisshort/flakehunter.git
cd flakehunter
go test ./...
go build -o flakehunter ./cmd/flakehunter
```

Windows:

```powershell
git clone https://github.com/edisshort/flakehunter.git
cd flakehunter
.\scripts\check.ps1
```

Optional demo suite:

```bash
pip install -r testdata/flaky_suite/requirements.txt
```

## Quick start

```bash
# Help
./flakehunter help

# Run the intentional flaky demo (pytest required)
./flakehunter run --runs 20 --min-outcomes 5 --workdir testdata/flaky_suite -- python -m pytest -q

# Report / inspect / list
./flakehunter report --workdir testdata/flaky_suite
./flakehunter inspect test_coin_flip --workdir testdata/flaky_suite
./flakehunter list --workdir testdata/flaky_suite

# Compare sequential vs parallel experiments
./flakehunter run --runs 12 --parallel 4 --workdir testdata/flaky_suite -- python -m pytest -q
./flakehunter compare <seq-id> <par-id> --workdir testdata/flaky_suite
```

PowerShell: use `.\flakehunter.exe` instead of `./flakehunter`.

## Commands

| Command | Purpose |
|---------|---------|
| `run` | Repeat a test command |
| `report` | Terminal summary + `analysis.json` + `report.json` |
| `inspect` | Per-test history and evidence |
| `compare` | Contrast two experiments |
| `list` | List saved experiments |
| `version` / `help` | Meta |

### Useful `run` flags

| Flag | Meaning |
|------|---------|
| `--runs N` | Required; number of executions |
| `--parallel N` | Max concurrent workers (default 1) |
| `--timeout 2m` | Per-run timeout |
| `--workdir PATH` | Working directory + storage root |
| `--order default\|random` | Random needs pytest-randomly |
| `--seed N` | Base seed for random order (`seed+i` per run) |
| `--min-outcomes N` | Classification threshold (default 5) |

## Classification

| Class | Meaning |
|------|---------|
| `stable_passing` | Enough decisive runs, never failed |
| `consistently_failing` | Enough decisive runs, never passed |
| `flaky` | Mixed pass and fail/error |
| `insufficient_evidence` | Below `--min-outcomes` |
| `inconsistent_presence` | Missing from some collected runs |

Skips do not count as pass or fail. Reports show **evidence**, not proven root causes.

## Measured demo results

From a real local run (2026-11-14). Details: [docs/VALIDATION.md](docs/VALIDATION.md).

| Test | Sequential (20) | Parallel 4 (12) |
|------|-----------------|-----------------|
| `test_always_passes` | stable | stable |
| `test_always_fails` | consistently failing 100% | 100% |
| `test_coin_flip` | **flaky 30%** | flaky (sampling variance) |
| `test_shared_counter_increment` | stable 0% | **flaky 58%** |

These are not performance benchmarks; re-measure on your machine.

## Project layout

```text
cmd/flakehunter/     # CLI entry
internal/            # engine, store, junit, analysis, report, ...
testdata/flaky_suite # intentional flakes for demos
docs/                # architecture, design, validation, samples
.github/workflows/   # CI
```

## Development

```bash
make check    # fmt + vet + test + race (Unix)
make build
```

CI runs on Ubuntu and Windows: format, `go vet`, `go test`, `go test -race`, build.

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Known limitations

- JUnit document order is best-effort, not a guaranteed execution trace
- `--order random` requires a compatible pytest plugin
- Parallel failure deltas are association evidence, not automatic concurrency proofs
- Local CLI only — not a secure multi-tenant execution service
- MVP runner focus is pytest; other runners need adapters

## Security

Treat commands and repositories as potentially untrusted. Do not run FlakeHunter as a privileged service that executes arbitrary remote commands.

## License

[MIT](LICENSE)
