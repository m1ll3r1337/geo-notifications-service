lint:
	golangci-lint run

build:
	go build -o ./bin/ ./cmd/api/main.go

run: build
	./bin/main
