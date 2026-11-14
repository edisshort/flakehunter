# Intentional flaky suite (Phase 8)

Small pytest project used to validate FlakeHunter. Failures are intentional.

| Test | Expected FlakeHunter class |
|------|----------------------------|
| `test_always_passes` | stable_passing |
| `test_always_fails` | consistently_failing |
| `test_coin_flip` | flaky (probabilistic) |
| `test_shared_file_*` | often flaky under `--parallel` |
| `test_order_*` | may be stable in one process; useful with order experiments |
| `test_timing_window` | sometimes flaky on loaded machines |

## Run directly

```powershell
pip install -r requirements.txt
python -m pytest -q
```

## Run via FlakeHunter

From the repo root:

```powershell
go build -o flakehunter.exe ./cmd/flakehunter
.\flakehunter.exe run --runs 20 --min-outcomes 5 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe report
.\flakehunter.exe inspect test_coin_flip
```

Compare sequential vs parallel:

```powershell
.\flakehunter.exe run --runs 12 --parallel 1 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe run --runs 12 --parallel 4 --workdir testdata/flaky_suite -- python -m pytest -q
.\flakehunter.exe compare <seq-id> <par-id>
```
