# Contributing

## Setup

1. Install Go 1.22+ (repo developed with Go 1.27)
2. Clone this repository
3. Run checks:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build -o flakehunter ./cmd/flakehunter
```

On Windows (PowerShell):

```powershell
.\scripts\check.ps1
```

## Demo suite (optional)

```bash
pip install -r testdata/flaky_suite/requirements.txt
./flakehunter run --runs 10 --workdir testdata/flaky_suite -- python -m pytest -q
```

## Guidelines

- Prefer the Go standard library
- Keep analysis deterministic and unit-tested
- Do not add LLM / paid AI dependencies
- Phrase findings as evidence, not absolute root cause
- Never invent benchmark numbers in docs
