.PHONY: dev-up dev-down dev-seed dev-produce dev-lineage \
        build test lint fmt \
        gateway-build contracts-build masking-build \
        lineage-build analyzer-build \
        help

DOCKER_COMPOSE := docker compose -f infra/docker/docker-compose.yml

# ── Local Development ────────────────────────────────────────────────────────

dev-up:
	$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "  Kafka:            localhost:9092"
	@echo "  Schema Registry:  localhost:8081"
	@echo "  Marquez:          http://localhost:5000"
	@echo "  Marquez UI:       http://localhost:3001"
	@echo "  Grafana:          http://localhost:3000  (admin / mzigo-dev)"
	@echo "  Prometheus:       http://localhost:9090"
	@echo "  MinIO:            http://localhost:9001  (mzigo-dev / mzigo-dev-secret)"
	@echo ""

dev-down:
	$(DOCKER_COMPOSE) down

dev-destroy:
	$(DOCKER_COMPOSE) down -v

dev-logs:
	$(DOCKER_COMPOSE) logs -f

dev-seed:
	@echo "loading example contracts and schemas..."
	go run ./tools/seed/main.go

dev-produce:
	@echo "running example producer against local gateway..."
	cd examples/producer-python && python produce.py

dev-lineage:
	open http://localhost:3001

dev-status:
	$(DOCKER_COMPOSE) ps

# ── Build ────────────────────────────────────────────────────────────────────

build: gateway-build contracts-build masking-build

gateway-build:
	cd services/gateway && go build -o bin/gateway ./cmd/gateway

contracts-build:
	cd services/contracts && go build -o bin/contracts ./cmd/contracts

masking-build:
	cd services/masking && go build -o bin/masking ./cmd/masking

# ── Test ─────────────────────────────────────────────────────────────────────

test: test-go test-python

test-go:
	cd services/gateway && go test ./... -race -count=1
	cd services/contracts && go test ./... -race -count=1
	cd services/masking && go test ./... -race -count=1

test-python:
	cd services/lineage && python -m pytest tests/ -v
	cd services/analyzer && python -m pytest tests/ -v

# ── Lint / Format ────────────────────────────────────────────────────────────

lint: lint-go lint-python

lint-go:
	cd services/gateway && golangci-lint run ./...
	cd services/contracts && golangci-lint run ./...
	cd services/masking && golangci-lint run ./...

lint-python:
	cd services/lineage && ruff check .
	cd services/analyzer && ruff check .

fmt:
	cd services/gateway && gofmt -w .
	cd services/contracts && gofmt -w .
	cd services/masking && gofmt -w .
	cd services/lineage && ruff format .
	cd services/analyzer && ruff format .

# ── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "Mzigo Development"
	@echo ""
	@echo "  make dev-up        Start local infrastructure stack"
	@echo "  make dev-down      Stop local infrastructure stack"
	@echo "  make dev-destroy   Stop and remove all volumes"
	@echo "  make dev-seed      Load example contracts and schemas"
	@echo "  make dev-produce   Run example producer"
	@echo "  make dev-lineage   Open Marquez UI"
	@echo ""
	@echo "  make build         Build all Go services"
	@echo "  make test          Run all tests"
	@echo "  make lint          Lint all code"
	@echo "  make fmt           Format all code"
	@echo ""
