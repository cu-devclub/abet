.PHONY: help build run test test-unit test-integration mock sqlc lint trivy pre-commit docker-build tidy clean migrate-new

APP_NAME := main-service
BINARY_PATH := bin/$(APP_NAME)
MAIN_PATH := ./cmd/api
PORT_FILES := $(wildcard internal/core/port/*.go)

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

tidy: ## Download and tidy Go dependencies
	go mod tidy
	go mod download

build: ## Build the Go API binary
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_PATH) $(MAIN_PATH)

run: ## Run the API service locally
	go run $(MAIN_PATH)

test: ## Run all tests (go test works on every platform)
	go test ./... -count=1

test-unit: ## Run unit tests only
	go test ./internal/... -count=1

test-integration: ## Run integration tests only
	go test ./tests/... -count=1

mock: ## Generate gomock mocks for every port file
	@for f in $(PORT_FILES); do \
		n=$$(basename $$f .go); \
		go run go.uber.org/mock/mockgen@v0.6.0 -source=$$f -destination=internal/core/port/mocks/mock_$$n.go -package=mocks; \
	done

sqlc: ## Generate database code using sqlc (docker)
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:1.29.0 generate

migrate-new: ## Create a migration pair: make migrate-new name=add_foo
	@test -n "$(name)" || (echo "usage: make migrate-new name=<snake_case>" && exit 1)
	@n=$$(printf "%06d" $$(( $$(ls db/migrations/*.up.sql 2>/dev/null | wc -l) + 1 ))); \
	touch db/migrations/$${n}_$(name).up.sql db/migrations/$${n}_$(name).down.sql; \
	echo "created db/migrations/$${n}_$(name).{up,down}.sql"

lint: ## Run golangci-lint
	golangci-lint run ./...

trivy: ## Scan repository vulnerabilities and misconfigurations using Trivy
	trivy fs --severity HIGH,CRITICAL --scanners vuln,misconfig,secret .

pre-commit: ## Run all pre-commit hooks
	pre-commit run --all-files

docker-build: ## Build the production Docker image
	docker build -t $(APP_NAME):latest .

clean: ## Clean build artifacts
	rm -rf bin/ tmp/ coverage.out
