# Roadmap

## Done (MVP)

- [x] CLI foundation
- [x] Subprocess execution with timeouts
- [x] pytest JUnit collection
- [x] Flakiness classification
- [x] Bounded parallel worker pool
- [x] Order/seed variation + compare
- [x] report / inspect / full JSON artifacts
- [x] Intentional flaky suite + measured validation
- [x] CI (fmt, vet, test, race)

## Next

- [x] Vitest / `npm test` JUnit collection
- [x] Lightweight install docs + release binaries workflow
- [ ] Additional runners (Go `go test` JSON, Jest)
- [ ] Quarantine list export (CI YAML snippets)
- [ ] Optional SQLite backend for large histories
- [ ] Richer duration anomaly stats
- [ ] Real open-source repo case studies (with permission)
- [ ] Point at QSV / other personal projects for field testing

## Non-goals (near term)

- Hosted multi-tenant runner
- LLM-based root-cause guessing
- Full web UI
