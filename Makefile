.PHONY: all build test race vet fmt check demo clean

all: check build

build:
	go build -o flakehunter ./cmd/flakehunter

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:" && gofmt -l . && exit 1)
	go vet ./...
	go test ./...
	go test -race ./...

demo: build
	./flakehunter run --runs 10 --min-outcomes 5 --workdir testdata/flaky_suite -- python -m pytest -q

clean:
	rm -f flakehunter flakehunter.exe
