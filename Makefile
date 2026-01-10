.PHONY: lint build run clean

lint:
	golangci-lint run

build:
	go build -o ./bin/main ./cmd/api/main.go

run: build
	./bin/main
