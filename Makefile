BINARY           := ps-top
GOLANGCI_VERSION := v2.15.0

.PHONY: all build test race vet fmt fmt-check lint lint-install tidy tidy-check cover check clean help \
	integration-up integration-load integration-down \
	repl-up repl-load repl-capture repl-states repl-down repl-matrix

all: build

build: ## Build the binary
	go build -o $(BINARY) .

test: ## Run unit tests
	go test ./...

race: ## Run tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Run gofmt in place
	gofmt -s -w .

fmt-check: ## Fail if any file needs gofmt
	@echo "gofmt -s -l ."
	@out=$$(gofmt -s -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found: run 'make lint-install'"; exit 1; }
	@have=$$(golangci-lint version --short); want=$(GOLANGCI_VERSION:v%=%); \
	if [ "$$(printf '%s\n%s\n' "$$have" "$$want" | sort -V | head -n1)" != "$$want" ]; then \
		echo "WARNING: golangci-lint $$have older than pinned $$want: run 'make lint-install'"; \
	fi
	golangci-lint run ./...

lint-install: ## Install pinned golangci-lint
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

tidy: ## Run go mod tidy
	go mod tidy

tidy-check: ## Fail if go mod tidy would change go.mod or go.sum
	go mod tidy -diff

cover: ## Show test coverage per function
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

check: fmt-check vet lint test tidy-check ## Run all CI checks

clean: ## Remove build outputs
	rm -f $(BINARY) coverage.out

# Integration harness (test/integration/README.md). Settings are passed as
# environment variables, e.g. make repl-up CONTAINER_IMAGE=mysql:8.4 GTID=on.
INTEGRATION := test/integration
INTEGRATION_COMPOSE := docker compose -f $(INTEGRATION)/docker-compose.yml $(if $(COMPOSE_EXTRA_FILE),-f $(COMPOSE_EXTRA_FILE))

integration-up: ## Start the single test server and wait until healthy
	$(INTEGRATION_COMPOSE) up -d --wait

integration-load: ## Run the load against the single test server
	$(INTEGRATION)/load/run.sh

integration-down: ## Stop the single test server and remove its data
	$(INTEGRATION_COMPOSE) down -v

repl-up: ## Start a source and two replicas and set up replication
	$(INTEGRATION)/replication.sh up

repl-load: ## Write on the replication source for LOAD_SECONDS
	$(INTEGRATION)/replication.sh load

repl-capture: ## Show the processlists of the source and replicas
	$(INTEGRATION)/replication.sh capture

repl-states: ## Count processlist states over SAMPLES samples
	$(INTEGRATION)/replication.sh states

repl-down: ## Stop the source and replicas and remove their data
	$(INTEGRATION)/replication.sh down

repl-matrix: ## Run the replication harness over all versions and settings
	$(INTEGRATION)/replication-matrix.sh

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "%-17s %s\n", $$1, $$2}'
