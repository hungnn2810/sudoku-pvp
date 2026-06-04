.DEFAULT_GOAL := build

.PHONY: docker-up docker-down docker-logs build run test test-integration generate lint tidy

docker-up:
	docker compose -f deployments/docker/compose.yml up -d

docker-down:
	docker compose -f deployments/docker/compose.yml down

docker-logs:
	docker compose -f deployments/docker/compose.yml logs -f

build:
	go build -o bin/api ./cmd/api/...

run:
	go run ./cmd/api/...

test:
	go test ./... -short -count=1

test-integration:
	go test ./... -count=1 -race -timeout 120s

generate:
	bash scripts/generate.sh

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
