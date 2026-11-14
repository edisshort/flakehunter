# Design decisions

## Why Go?

- Strong standard library for subprocesses (`os/exec`) and concurrency
- Single static binary for a CLI tool
- Race detector and `testing` package fit the problem

## Why JSON files, not a database?

- Zero setup for developers trying the tool
- Artifacts are inspectable and easy to gitignore
- Enough for local MVP experiments

Trade-off: large suites with thousands of runs may later want SQLite.

## Why pytest + JUnit first?

- Common Python ecosystem
- Machine-readable per-test results without proprietary formats
- Process exit code alone is not enough for flakiness

## Why no LLM / ML?

- Results must be reproducible and explainable
- Interview / engineering signal: systems + stats, not prompt wrappers
- Evidence language avoids false root-cause claims

## Classification rules

- Skips are not decisive
- Errors count with failures for failure rate
- Default `--min-outcomes 5` before calling a test stable/flaky/consistent
- 30/30 failures ⇒ consistently failing, **not** flaky

## Parallel failures ≠ concurrency bug

Higher failure rate under `--parallel` is **association evidence**. The intentional shared-file fixture demonstrates the signal; real projects still need human investigation.

## Security posture

- Local CLI only; the user chooses which command to run
- Not a multi-tenant execution service
- Do not run untrusted repos with elevated privileges
- Absolute JUnit paths avoid writing outside the experiment directory when cwd differs

## Alternatives considered

| Topic | Chosen | Rejected for MVP |
|-------|--------|------------------|
| CLI parsing | Hand-rolled | Cobra (add later if needed) |
| Storage | JSON | SQLite |
| UI | Terminal | React dashboard |
| Random order | pytest-randomly seed flag | Re-implementing collection |
