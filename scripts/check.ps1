# Local quality gate (Windows)
$ErrorActionPreference = "Stop"
$env:Path = "C:\Program Files\Go\bin;" + $env:Path

Write-Host "== gofmt =="
$bad = gofmt -l .
if ($bad) {
    Write-Host "Need gofmt:"
    Write-Host $bad
    exit 1
}

Write-Host "== go vet =="
go vet ./...

Write-Host "== go test =="
go test ./...

Write-Host "== go test -race =="
go test -race ./...

Write-Host "== build =="
go build -o flakehunter.exe ./cmd/flakehunter

Write-Host "OK"
