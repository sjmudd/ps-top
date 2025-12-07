PROJECT_NAME := "ps-top"
PKG := "github.com/sjmudd/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)

.PHONY: all dep build clean test coverage coverhtml lint fmt fmt-check lint-install ci-check tidy tidy-check

all: build

fmt: ## Run gofmt in-place
	@gofmt -s -w .

fmt-check: ## Check for gofmt changes (fail if files need formatting)
	@unformatted=$(gofmt -s -l .) ; if [ -n "$$unformatted" ]; then echo "gofmt needs to be run on:"; echo "$$unformatted"; exit 1; fi

lint-install: ## Install golangci-lint (v1.59.0)
	@echo "Installing golangci-lint v1.59.0..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.59.0

lint: ## Lint the files with golangci-lint (requires lint-install)
	@which golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found; run 'make lint-install'"; exit 1; }
	@golangci-lint run ./...


test: ## Run unittests
	@go test -short ${PKG_LIST}

race: dep ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: dep ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

coverage: ## Generate global code coverage report
	./tools/coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./tools/coverage.sh html;

dep: ## Get the dependencies
	@go get -v ./...

ci-check: fmt-check lint test tidy-check ## Run CI-like checks locally
	@echo "ci-check completed"

tidy: ## Run go mod tidy (updates go.mod & go.sum)
	@go mod tidy

tidy-check: ## Ensure go.mod and go.sum are tidy (fail if changes)
	@orig=$$(mktemp) ; git ls-files -- others --ignored --exclude-standard >/dev/null 2>&1 || true ; git rev-parse --verify HEAD >/dev/null 2>&1 || true ; git status --porcelain >/dev/null 2>&1 || true ; git diff --quiet || true ; go mod tidy ; if [ -n "$$(git status --porcelain)" ]; then echo "go.mod or go.sum changed after go mod tidy; please run 'go mod tidy'"; git --no-pager status --porcelain; git --no-pager diff; git checkout -- go.mod go.sum || true; exit 1; fi ; rm -f $$orig

build: dep ## Build the binary file
	@go build -v $(PKG)

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
