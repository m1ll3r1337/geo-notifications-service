.PHONY: lint build run migrate-up migrate-down migrate-version run-webhook-server

GEO_DB_URL?=postgres://postgres:postgres@localhost:5432/geo_not?sslmode=disable

lint:
	golangci-lint run

build:
	go build -o ./bin/main ./cmd/api/main.go

run: build
	./bin/main

migrate-up:
	migrate -path ./migrations -database "$(GEO_DB_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(GEO_DB_URL)" down

migrate-version:
	migrate -path ./migrations -database "$(GEO_DB_URL)" version

test-integration:
	go test -tags=integration ./... -count=1

run-webhook-server:
	WEBHOOK_URL=http://localhost:9090/webhook go run cmd/webhook-test/main.go