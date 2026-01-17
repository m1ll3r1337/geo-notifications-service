FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/webhook-test ./cmd/webhook-test/main.go

FROM debian:bookworm-slim AS api
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM debian:bookworm-slim AS webhook-test
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/webhook-test /app/webhook-test
EXPOSE 9090
ENTRYPOINT ["/app/webhook-test"]   