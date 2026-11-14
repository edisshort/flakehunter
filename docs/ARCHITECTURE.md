# Architecture

FlakeHunter is a **local CLI** that orchestrates repeated test runs, persists structured results as JSON, and classifies instability with deterministic statistics.

No LLM APIs. No database in the MVP.

## Component diagram

```text
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                           │
│   run | report | inspect | compare | list | help | version  │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                   Experiment Manager                        │
│   create ID, schedule N runs, sequential or worker pool     │
└───────────────┬─────────────────────────────┬───────────────┘
                │                             │
                ▼                             ▼
┌───────────────────────────┐   ┌─────────────────────────────┐
│     Execution Engine      │   │   Result Collector          │
│  os/exec + context        │──▶│  pytest JUnit XML parser    │
│  timeout, logs, exit code │   │  per-test pass/fail/skip    │
└───────────────────────────┘   └──────────────┬──────────────┘
                                               │
                                               ▼
                                 ┌─────────────────────────────┐
                                 │   Storage (.flakehunter/)   │
                                 │   meta / executions / JSON  │
                                 └──────────────┬──────────────┘
                                                │
                                                ▼
                                 ┌─────────────────────────────┐
                                 │  Analysis + Report          │
                                 │  classify, evidence, inspect│
                                 └─────────────────────────────┘
```

## Package map

| Package | Responsibility |
|---------|----------------|
| `cmd/flakehunter` | Process entry (`main`) |
| `internal/cli` | Flags, commands, exit codes |
| `internal/experiment` | Multi-run orchestration + worker pool |
| `internal/engine` | Single subprocess lifecycle |
| `internal/store` | JSON file persistence |
| `internal/junit` | JUnit XML → `TestCase` |
| `internal/pytestutil` | pytest argv / seed helpers |
| `internal/analysis` | Stats, classification, compare |
| `internal/report` | Terminal + full JSON + inspect |
| `internal/model` | Shared types |

## Data on disk

```text
.flakehunter/
  experiments/<id>/
    meta.json
    analysis.json
    report.json
    inspections/<test>.json
    executions/0001/
      result.json
      stdout.log
      stderr.log
      junit.xml
```

When `--workdir` is set, this tree lives under that directory.

## Concurrency model (Phase 5)

- Job channel of run indexes → up to `--parallel` workers
- Each run writes only to its own `executions/NNNN/` folder
- Logging uses a mutex; cancel via `context` (Ctrl+C)

## Extension points

1. **New runners** — add a parser beside `junit` and argv helpers beside `pytestutil`
2. **Storage** — swap `store` for SQLite later without changing analysis
3. **Analysis** — keep classification pure functions for easy unit tests
