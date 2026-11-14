BINARY  := helm-values-tidy
BIN_DIR := bin
GO      ?= go

# Built with the current Go toolchain, so it can read packages from new Go
# releases (a globally installed binary built with an older Go can't).
GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT ?= $(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Remote chart used by `make demo-remote`.
REMOTE_CHART ?= oci://registry-1.docker.io/bitnamicharts/nginx

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary into ./bin
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

.PHONY: install
install: ## Install the binary into $GOBIN
	$(GO) install .

.PHONY: test
test: ## Run unit tests
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests with coverage and print a summary
	@mkdir -p $(BIN_DIR)
	$(GO) test -race -coverprofile=$(BIN_DIR)/cover.out ./...
	$(GO) tool cover -func=$(BIN_DIR)/cover.out | tail -1

.PHONY: fmt
fmt: ## Format all Go code
	gofmt -w .

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	$(GO) mod tidy

.PHONY: check
check: vet lint test ## Run vet, lint and tests (use before committing)
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)

.PHONY: demo
demo: ## Run against the local test chart
	$(GO) run . -f testdata/values.yaml testdata/parent

.PHONY: demo-remote
demo-remote: ## Run against a remote OCI chart with the test values (needs network)
	$(GO) run . -f testdata/values.yaml $(REMOTE_CHART)

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
