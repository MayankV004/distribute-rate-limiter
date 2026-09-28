SHELL := /usr/bin/bash
GO    ?= go
PKG   := ./...
CFG   ?= configs/gateway.dev.yaml

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build gateway and dummy backend into bin/
	$(GO) build -o bin/gateway ./cmd/gateway
	$(GO) build -o bin/backend ./cmd/backend

.PHONY: run
run: ## Run the gateway with the dev config
	$(GO) run ./cmd/gateway -config $(CFG)

.PHONY: test
test: ## Unit tests
	$(GO) test $(PKG)

.PHONY: race
race: ## Concurrency tests under the race detector (non-negotiable in CI)
	$(GO) test -race -count=1 $(PKG)

.PHONY: integration
integration: ## Integration tests against a real Redis (testcontainers)
	$(GO) test -tags=integration -count=1 ./test/integration/...

.PHONY: bench
bench: ## Benchmarks, no test execution
	$(GO) test -run='^$$' -bench=. -benchmem $(PKG)

.PHONY: vet
vet: ## go vet
	$(GO) vet $(PKG)

.PHONY: lint
lint: ## golangci-lint (install: https://golangci-lint.run)
	golangci-lint run

.PHONY: tidy
tidy: ## Sync go.mod/go.sum
	$(GO) mod tidy

.PHONY: up
up: ## Bring up the multi-node stack (nginx + 3 gateways + redis + prom + grafana)
	docker compose -f deployments/docker-compose.yaml up --build -d

.PHONY: down
down: ## Tear the stack down, including volumes
	docker compose -f deployments/docker-compose.yaml down -v

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/
