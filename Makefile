all: test lint

test:
	go test -race ./...

lint: .golangci.yml
	CGO_ENABLED=0 golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: all lint test
