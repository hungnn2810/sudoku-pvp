.DEFAULT_GOAL := build

.PHONY: docker-up docker-down docker-logs build run test test-integration test-coverage generate lint fmt check verify-no-fmt-println tidy

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
	go test ./... -tags integration -count=1 -race -timeout 120s

test-coverage:
	go test ./... -short -coverprofile=coverage.out -covermode=atomic -count=1
	go tool cover -func=coverage.out | tail -1

generate:
	bash scripts/generate.sh

lint:
	golangci-lint run ./...

fmt:
	gofmt -w ./internal/ ./cmd/

# check runs vet and a fallback fmt.Println guard when golangci-lint is not installed.
check:
	go vet ./...
	@grep -rn --include="*.go" "fmt\.Println\|fmt\.Printf\|fmt\.Print" internal/ cmd/ | grep -v "_test\.go" | grep -v "^Binary" | grep -v "^$$" && echo "FAIL: fmt.Println found in production code" && exit 1 || echo "OK: no fmt.Println in production code"

# verify-no-fmt-println enforces that no production Go code uses fmt.Print* functions.
verify-no-fmt-println:
	@grep -rn --include="*.go" "fmt\.Println\|fmt\.Printf\|fmt\.Print" internal/ cmd/ | grep -v "_test\.go" | grep -v "^Binary" | grep -v "^$$" && echo "FAIL: fmt.Println found in production code" && exit 1 || echo "OK: no fmt.Println in production code"

tidy:
	go mod tidy
