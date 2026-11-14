# Install FlakeHunter on any machine

Yes — you can install a **lightweight binary** without copying the whole repo every time.

FlakeHunter is a single Go CLI. Other people only need the executable (plus whatever test runner their project uses: pytest, npm/vitest, etc.).

## Option A — `go install` (simplest if Go is installed)

```bash
go install github.com/edisshort/flakehunter/cmd/flakehunter@latest
```

Then ensure Go’s bin dir is on `PATH` (often `~/go/bin` or `%USERPROFILE%\go\bin`).

```bash
flakehunter version
flakehunter run --runs 10 --workdir /path/to/project -- npm test
```

## Option B — Download a release binary (no Go required)

1. Open [GitHub Releases](https://github.com/edisshort/flakehunter/releases)
2. Download the build for your OS (Windows `.exe`, Linux, or macOS)
3. Put it on your `PATH` or run it by full path

```powershell
.\flakehunter.exe version
.\flakehunter.exe run --runs 10 --workdir C:\path\to\qsb-nse -- npm test
```

Creating a release tag triggers the binary build workflow (see `.github/workflows/release.yml`).

## Option C — Build from source

```bash
git clone https://github.com/edisshort/flakehunter.git
cd flakehunter
go build -o flakehunter ./cmd/flakehunter
```

## What still must exist on the target machine?

| Need | Why |
|------|-----|
| FlakeHunter binary | The orchestrator |
| Project’s test tools | e.g. Node+npm for Vitest, or Python+pytest |
| Project dependencies | `npm install` / `pip install` as usual |

FlakeHunter does **not** bundle Node or Python. It only launches the command you pass after `--`.

## Not an npm package (by design)

Publishing FlakeHunter as `npm i -g flakehunter` is possible later as a thin wrapper, but the native lightweight form for a Go CLI is **`go install` or a release `.exe`**. That keeps it fast and dependency-free.
