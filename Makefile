SHELL := /bin/bash

.PHONY: lint test fmt tidy ci

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...
	@gofumpt -l -w . || true

test:
	@go test ./...

tidy:
	@go mod tidy -v

ci: tidy fmt lint test
