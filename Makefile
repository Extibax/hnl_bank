.PHONY: up down build test

# Docker based helpers (no local Go required)
GO_IMAGE  ?= golang:1.26
MOD_CACHE ?= $(HOME)/go/pkg/mod

up:
	docker compose up -d --build

down:
	docker compose down -v

build:
	docker compose build

# Runs the Go build + vet + service unit tests inside an ephemeral golang container.
# This works even on machines without a local Go toolchain (e.g. Windows + WSL2).
test:
	docker run --rm -v "$(CURDIR)/backend:/src" -v "$(MOD_CACHE):/go/pkg/mod" -w /src $(GO_IMAGE) sh -c "go build ./... && go vet ./... && go test ./internal/service/..."
