all: test lint

test:
	go test -race ./...

lint: .golangci.yml
	CGO_ENABLED=0 golangci-lint run

# Run locally against the grol-dev bot: GROL_DEV_TOKEN=... [DISCORD_BOT_ADMIN=youruserid] make dev
GROL_DISABLE_AUTOSAVE ?= 1
dev:
	@test -n "$(GROL_DEV_TOKEN)" || (echo "GROL_DEV_TOKEN must be set to the grol-dev bot token"; exit 1)
	@DISCORD_BOT_TOKEN=$(GROL_DEV_TOKEN) GOMEMLIMIT=1GiB GROL_DISABLE_AUTOSAVE=$(GROL_DISABLE_AUTOSAVE) go run .

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml

.PHONY: all lint test dev
