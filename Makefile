.PHONY: all dev build test lint format precommit swag

dev:
	go run cmd/server/main.go

build:
	mkdir -p bin
	go build -o bin/auth-server ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run

format:
	gofmt -w $(shell go list -f '{{.Dir}}' ./...)

precommit:
	make format
	make lint
	make test

swag:
	swag init -g cmd/server/main.go

all:
	precommit


