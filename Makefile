APP_NAME := google_maps_scraper
VERSION := 1.10.0
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS := -X 'github.com/gosom/google-maps-scraper/runner.Version=$(VERSION)' \
           -X 'github.com/gosom/google-maps-scraper/runner.Commit=$(COMMIT_HASH)' \
           -X 'github.com/gosom/google-maps-scraper/runner.BuildDate=$(BUILD_DATE)' \
           -w -s

# Database configuration (override with environment variables)
POSTGRES_USER ?= gmaps
POSTGRES_PASSWORD ?= gmaps_secret
POSTGRES_DB ?= gmaps
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Server configuration
MANAGER_ADDR ?= :8080
MANAGER_URL ?= http://localhost:8080
WORKER_CONCURRENCY ?= 4

default: help

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# ===========================================
# Development
# ===========================================

vet: ## runs go vet
	go vet ./...

format: ## runs go fmt
	gofmt -s -w .

test: ## runs the unit tests
	go test -v -race -timeout 5m ./...

test-cover: ## outputs the coverage statistics
	go test -v -race -timeout 5m ./... -coverprofile coverage.out
	go tool cover -func coverage.out
	rm coverage.out

test-cover-report: ## an html report of the coverage statistics
	go test -v ./... -covermode=count -coverpkg=./... -coverprofile coverage.out
	go tool cover -html coverage.out -o coverage.html
	open coverage.html

vuln: ## runs vulnerability checks
	go tool govulncheck -C . -show verbose -format text -scan symbol ./...

lint: ## runs the linter
	go tool golangci-lint -v run ./...

# ===========================================
# Build
# ===========================================

build: ## builds the application (default: playwright)
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) .

build-rod: ## builds the application with go-rod browser engine
	go build -tags rod -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-rod .

cross-compile: ## cross compiles the application
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-darwin-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-windows-amd64.exe

cross-compile-rod: ## cross compiles the application with go-rod
	GOOS=linux GOARCH=amd64 go build -tags rod -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-rod-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -tags rod -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-rod-darwin-amd64
	GOOS=windows GOARCH=amd64 go build -tags rod -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME)-${VERSION}-rod-windows-amd64.exe

# ===========================================
# Docker
# ===========================================

docker: ## builds docker image with playwright (default)
	docker build -t $(APP_NAME):$(VERSION) .

docker-rod: ## builds docker image with go-rod
	docker build -f Dockerfile.rod -t $(APP_NAME):$(VERSION)-rod .

# ===========================================
# Database
# ===========================================

db-start: ## starts PostgreSQL database using docker
	docker run -d --name gmaps-postgres \
		-e POSTGRES_USER=$(POSTGRES_USER) \
		-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
		-e POSTGRES_DB=$(POSTGRES_DB) \
		-p $(POSTGRES_PORT):5432 \
		-v gmaps-postgres-data:/var/lib/postgresql/data \
		postgres:15-alpine
	@echo "Waiting for database to be ready..."
	@sleep 3
	@echo "PostgreSQL started on port $(POSTGRES_PORT)"

db-stop: ## stops PostgreSQL database
	docker stop gmaps-postgres || true
	docker rm gmaps-postgres || true

db-logs: ## shows PostgreSQL logs
	docker logs -f gmaps-postgres

db-shell: ## opens psql shell
	docker exec -it gmaps-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

db-reset: db-stop db-start ## resets the database (stops, removes, starts fresh)

# ===========================================
# Manager/Worker Mode (New Architecture)
# ===========================================

run-manager: build ## runs the manager (API server, no scraping)
	./bin/$(APP_NAME) -manager -dsn "$(DATABASE_URL)" -addr "$(MANAGER_ADDR)"

run-manager-sqlite: build ## runs the manager with SQLite
	./bin/$(APP_NAME) -manager -addr "$(MANAGER_ADDR)"

run-worker: build ## runs a worker (connects to manager)
	./bin/$(APP_NAME) -worker -manager-url "$(MANAGER_URL)" -c $(WORKER_CONCURRENCY)

run-manager-rod: build-rod ## runs the manager with go-rod build
	./bin/$(APP_NAME)-rod -manager -dsn "$(DATABASE_URL)" -addr "$(MANAGER_ADDR)"

run-worker-rod: build-rod ## runs a worker with go-rod build
	./bin/$(APP_NAME)-rod -worker -manager-url "$(MANAGER_URL)" -c $(WORKER_CONCURRENCY)

# ===========================================
# Docker Compose
# ===========================================

up: ## starts manager + workers using docker-compose
	docker compose up -d db
	@echo "Waiting for database..."
	@sleep 5
	docker compose up -d manager
	@echo "Waiting for manager..."
	@sleep 3
	docker compose up -d worker
	@echo "Manager running at http://localhost:8080"
	@echo "Use 'make logs' to view logs"

up-scale: ## starts with multiple workers (usage: make up-scale WORKERS=3)
	docker compose up -d db
	@sleep 5
	docker compose up -d manager
	@sleep 3
	docker compose up -d --scale worker=$(WORKERS) worker
	@echo "Manager running at http://localhost:8080 with $(WORKERS) workers"

down: ## stops all docker-compose services
	docker compose down

logs: ## shows docker-compose logs
	docker compose logs -f

logs-manager: ## shows manager logs
	docker compose logs -f manager

logs-worker: ## shows worker logs
	docker compose logs -f worker

ps: ## shows running containers
	docker compose ps

# ===========================================
# Legacy Mode (Web UI + Scraping in one process)
# ===========================================

up-legacy: ## starts legacy web mode (UI + scraping together)
	docker compose --profile legacy up -d

run-web: build ## runs legacy web mode locally
	./bin/$(APP_NAME) -web -dsn "$(DATABASE_URL)"

# ===========================================
# Quick Start
# ===========================================

quick-start: db-start run-manager ## quick start: starts db and manager locally

quick-start-sqlite: run-manager-sqlite ## quick start with SQLite (no docker required)

dev: ## development mode: starts db, builds, and runs manager
	@make db-start 2>/dev/null || true
	@sleep 2
	@make run-manager
