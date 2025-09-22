SHELL := /usr/bin/env bash
GO    ?= go
GOFMT ?= gofmt
GOLANGCI_LINT ?= golangci-lint

MODULES := $(shell $(GO) list ./... 2>/dev/null)

# -race requires CGO on Windows; use plain tests locally unless you've enabled CGO + a C toolchain.
ifeq ($(OS),Windows_NT)
GO_TEST_FLAGS ?= -count=1
else
GO_TEST_FLAGS ?= -race -count=1
endif

.PHONY: help
help:
	@echo "FluxLens — common development targets"
	@echo ""
	@echo "  make tidy        Run 'go mod tidy'"
	@echo "  make fmt         Run gofmt across the repo"
	@echo "  make vet         Run 'go vet'"
	@echo "  make lint        Run golangci-lint"
	@echo "  make test        Run all unit tests (adds -race except on Windows; override with GO_TEST_FLAGS=)"
	@echo "  make cover       Run tests with coverage"
	@echo "  make build       Build all command binaries into ./bin"
	@echo "  make dev         Start local docker-compose stack"
	@echo "  make dev-down    Stop local docker-compose stack"
	@echo "  make synth       Run synthetic event generator against local stack"
	@echo "  make demo        End-to-end demo: bring up stack and run synth"
	@echo ""

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: fmt
fmt:
	$(GOFMT) -l -w $$(find . -name '*.go' -not -path './vendor/*')

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: test
test:
	$(GO) test $(GO_TEST_FLAGS) ./...

.PHONY: cover
cover:
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -func=coverage.txt | tail -1

.PHONY: build
build:
	mkdir -p bin
	$(GO) build -o bin/fluxlens-curator       ./cmd/curator
	$(GO) build -o bin/fluxlens-api-gateway   ./cmd/api-gateway
	$(GO) build -o bin/fluxlens-orchestrator  ./cmd/orchestrator
	$(GO) build -o bin/fluxlens-audit-writer  ./cmd/audit-writer
	$(GO) build -o bin/fluxlens-ingest-mysql  ./cmd/ingest-mysql
	$(GO) build -o bin/fluxlens-synth-source  ./cmd/synth-source

.PHONY: dashboard-install dashboard-dev dashboard-build
dashboard-install:
	cd dashboard && npm install --no-audit --no-fund

dashboard-dev:
	cd dashboard && npm run dev

dashboard-build:
	cd dashboard && npm run build

.PHONY: docker-build
docker-build:
	for cmd in curator api-gateway orchestrator audit-writer ingest-mysql synth-source; do \
	  docker build --build-arg CMD=$$cmd -t fluxlens/$$cmd:dev .; \
	done
	docker build -t fluxlens/dashboard:dev ./dashboard

.PHONY: dev
dev:
	docker compose up -d
	@echo "FluxLens dev stack is starting. Run 'make dev-status' to check health."

.PHONY: dev-status
dev-status:
	docker compose ps

.PHONY: dev-down
dev-down:
	docker compose down

.PHONY: dev-clean
dev-clean:
	docker compose down -v

.PHONY: synth
synth:
	$(GO) run ./cmd/synth-source --kafka localhost:9092 --rate 100 --source-count 20

.PHONY: demo
demo: dev
	@echo "Waiting 15s for Kafka to be ready..."
	@sleep 15
	$(MAKE) synth
