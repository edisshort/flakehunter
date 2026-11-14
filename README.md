# FlakeHunter

CLI tool that repeatedly runs automated tests, stores per-run results, and (in later phases) detects flaky tests.

**Status:** Phase 3 — sequential execution + pytest JUnit XML collection.

## Requirements

- Go 1.22+ (developed with Go 1.27)
- For pytest experiments: Python + pytest

## Quick start

```powershell
go test ./...
go build -o flakehunter.exe ./cmd/flakehunter

.\flakehunter.exe help
.\flakehunter.exe run --runs 5 -- go version
.\flakehunter.exe run --runs 5 --timeout 2m -- python -m pytest
.\flakehunter.exe list
```

Results are stored under `.flakehunter/experiments/<id>/`.

## Phases

| Phase | Focus |
|------|--------|
| 1 | CLI foundation |
| 2 | Subprocess execution engine (done) |
| 3 | pytest + JUnit XML collection (done) |
| 4 | Flakiness analysis |
| 5 | Parallel worker pool |
| 6+ | Order variation, richer reports, polish |

## License

MIT (to be confirmed)
