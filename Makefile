# Makefile for Family Service
# This Makefile provides targets for building, testing, and deploying the Family Service application.

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run
GOLINT=golangci-lint
GOVULNCHECK=$(GOCMD) run golang.org/x/vuln/cmd/govulncheck

# Application parameters
BINARY_NAME=family-service
MAIN_PATH=./cmd/server
BINARY_OUTPUT=./bin/$(BINARY_NAME)
DOCKER_IMAGE=family-service
DOCKER_TAG=latest

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Cross-compilation parameters
PLATFORMS=linux darwin windows
ARCHITECTURES=amd64 arm64

# GraphQL parameters
GQLGEN=go run github.com/99designs/gqlgen
GQLGEN_CONFIG=./gqlcfg.yml

# Migration parameters
MIGRATE_MONGO=go run $(MAIN_PATH)/migration/mongo/migrate_mongo.go
MIGRATE_POSTGRES=go run $(MAIN_PATH)/migration/postgresql/migrate_postgres.go

# Default target
.PHONY: all
all: help

# Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# Show help
.PHONY: help
help:
	@echo "Family Service Makefile"
	@echo "Usage:"
	@echo ""
	@echo "Build and Development:"
	@echo "  make init              - Initialize development environment"
	@echo "  make generate          - Generate GraphQL code"
	@echo "  make build             - Build the application"
	@echo "  make build-all         - Build the application for all platforms and architectures"
	@echo "  make run               - Run the application locally"
	@echo "  make dev               - Run the application with hot reloading"
	@echo "  make clean             - Remove build artifacts"
	@echo "  make tidy              - Tidy and verify Go modules"
	@echo "  make fmt               - Format code"
	@echo "  make deps              - Download dependencies"
	@echo "  make deps-upgrade      - Upgrade dependencies"
	@echo "  make deps-graph        - Generate dependency graph"
	@echo ""
	@echo "Testing:"
	@echo "  make test              - Run all tests"
	@echo "  make test-all          - Run all tests with coverage and generate a combined report"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-race         - Run tests with race detection"
	@echo "  make test-timeout      - Run tests with timeout"
	@echo "  make test-timeout-behavior - Run tests that verify timeout behavior"
	@echo "  make test-package      - Run tests for a specific package (PKG=./path/to/package)"
	@echo "  make test-package-coverage - Run tests with coverage for a specific package (PKG=./path/to/package)"
	@echo "  make test-run          - Run tests matching a specific pattern (PATTERN=TestName)"
	@echo "  make test-run-coverage - Run tests with coverage matching a specific pattern (PATTERN=TestName)"
	@echo "  make test-bench        - Run benchmarks"
	@echo ""
	@echo "Coverage:"
	@echo "  make test-coverage     - Generate test coverage report"
	@echo "  make test-coverage-view - View test coverage in browser"
	@echo "  make test-coverage-summary - Show test coverage summary"
	@echo "  make test-coverage-func - Show test coverage by function"
	@echo "  make test-coverage-func-sorted - Show test coverage by function, sorted"
	@echo ""
	@echo "Profiling:"
	@echo "  make profile-cpu       - Run CPU profiling"
	@echo "  make profile-mem       - Run memory profiling"
	@echo "  make profile-block     - Run block profiling"
	@echo "  make profile-mutex     - Run mutex profiling"
	@echo "  make profile-trace     - Run execution tracing"
	@echo "  make profile-all       - Run all profiling"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint              - Run linters"
	@echo "  make vuln-check        - Check for vulnerabilities in dependencies"
	@echo "  make pre-commit        - Run all pre-commit checks"
	@echo "  make verify-translations - Verify that all translation strings are used in the code"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs              - Generate and serve documentation"
	@echo "  make docs-pkgsite      - Generate and serve documentation with pkgsite"
	@echo "  make docs-static       - Generate static documentation"
	@echo ""
	@echo "Docker:"
	@echo "  make dockerfile        - Create a basic Dockerfile"
	@echo "  make docker            - Build Docker image"
	@echo "  make docker-run        - Run Docker container"
	@echo "  make docker-compose-up - Start all services with Docker Compose"
	@echo "  make docker-compose-down - Stop all services with Docker Compose"
	@echo "  make docker-compose-logs - Show logs for all services"
	@echo "  make docker-compose-ps - Show status of all services"
	@echo "  make docker-compose-restart - Restart all services"
	@echo "  make docker-compose-build - Build all services"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-mongo     - Run MongoDB migrations"
	@echo "  make migrate-postgres  - Run PostgreSQL migrations"
	@echo "  make migrate           - Run all migrations"
	@echo "  make drop-test-dbs     - Drop test databases"
	@echo "  make recreate-migrations - Recreate migration scripts"
	@echo "  make recreate-integration-tests - Recreate integration tests"
	@echo "  make recreate-all      - Recreate all integration tests and migration scripts"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy            - Deploy the application"
	@echo ""
	@echo "Configuration:"
	@echo "  make airconfig         - Create a basic .air.toml configuration file"
	@echo ""
	@echo "Help:"
	@echo "  make help              - Show this help message"
	@echo "  make version           - Show version information"

# Initialize development environment
.PHONY: init
init:
	@echo "Initializing development environment..."
	$(GOGET) -u github.com/99designs/gqlgen
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/cosmtrek/air
	$(GOGET) -u golang.org/x/vuln/cmd/govulncheck
	$(GOMOD) download
	@echo "Development environment initialized successfully"

# Generate GraphQL code
.PHONY: generate
generate:
	@echo "Generating GraphQL code..."
	go get github.com/99designs/gqlgen/codegen/config@v0.17.73
	go get github.com/99designs/gqlgen/internal/imports@v0.17.73
	go get github.com/99designs/gqlgen/api@v0.17.73
	go get github.com/99designs/gqlgen@v0.17.73
	cd internal/adapters/graphql && $(GQLGEN) generate --config $(GQLGEN_CONFIG)
	@echo "GraphQL code generated successfully"

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_OUTPUT) $(MAIN_PATH)
	@echo "Build successful: $(BINARY_OUTPUT)"

# Build the application for all platforms and architectures
.PHONY: build-all
build-all:
	@echo "Building $(BINARY_NAME) for all platforms..."
	$(foreach platform,$(PLATFORMS),\
		$(foreach arch,$(ARCHITECTURES),\
			$(eval os := $(platform))\
			$(eval ext := $(if $(filter windows,$(platform)),.exe,))\
			@echo "Building for $(os)/$(arch)..." && \
			mkdir -p bin/$(os)_$(arch) && \
			GOOS=$(os) GOARCH=$(arch) $(GOBUILD) $(LDFLAGS) -o bin/$(os)_$(arch)/$(BINARY_NAME)$(ext) $(MAIN_PATH) && \
			echo "Build successful: bin/$(os)_$(arch)/$(BINARY_NAME)$(ext)" ; \
		)\
	)
	@echo "All builds completed successfully"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests completed"

# Run all tests with coverage and generate a combined report
.PHONY: test-all
test-all:
	@echo "Running all tests with coverage..."
	mkdir -p ./coverage
	$(GOTEST) -v -coverprofile=./coverage/all.out ./...
	$(GOCMD) tool cover -html=./coverage/all.out -o ./coverage/all.html
	$(GOCMD) tool cover -func=./coverage/all.out
	@echo "All tests completed and coverage report generated: ./coverage/all.html"

# Run unit tests only
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short ./...
	@echo "Unit tests completed"

# Run integration tests only
.PHONY: test-integration
test-integration: drop-test-dbs
	@echo "Running integration tests..."
	$(GOTEST) -v -run Integration ./...
	@echo "Integration tests completed"

# Recreate integration tests
.PHONY: recreate-integration-tests
recreate-integration-tests: drop-test-dbs
	@echo "Recreating integration tests..."
	$(GOTEST) -v -run Integration ./internal/adapters/mongodb/...
	$(GOTEST) -v -run Integration ./internal/adapters/postgres/...
	@echo "Integration tests recreated successfully"

# Generate test coverage report
.PHONY: test-coverage
test-coverage:
	@echo "Generating test coverage report..."
	mkdir -p ./coverage
	$(GOTEST) -coverprofile=./coverage/coverage.out ./...
	$(GOCMD) tool cover -html=./coverage/coverage.out -o ./coverage/coverage.html
	@echo "Coverage report generated: ./coverage/coverage.html"

# Show test coverage summary
.PHONY: test-coverage-summary
test-coverage-summary:
	@echo "Generating test coverage summary..."
	$(GOTEST) -cover ./...
	@echo "Coverage summary completed"

# Show test coverage by function
.PHONY: test-coverage-func
test-coverage-func:
	@echo "Generating test coverage by function..."
	mkdir -p ./coverage
	$(GOTEST) -coverprofile=./coverage/coverage.out ./...
	$(GOCMD) tool cover -func=./coverage/coverage.out
	@echo "Function coverage completed"

# Show test coverage by function, sorted by coverage percentage
.PHONY: test-coverage-func-sorted
test-coverage-func-sorted:
	@echo "Generating test coverage by function (sorted)..."
	mkdir -p ./coverage
	$(GOTEST) -coverprofile=./coverage/coverage.out ./...
	$(GOCMD) tool cover -func=./coverage/coverage.out | sort -k 3 -n
	@echo "Sorted function coverage completed"

# View test coverage in browser
.PHONY: test-coverage-view
test-coverage-view: test-coverage
	@echo "Opening coverage report in browser..."
	$(GOCMD) tool cover -html=./coverage/coverage.out
	@echo "Coverage report opened in browser"

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...
	@echo "Race detection tests completed"

# Run benchmarks
.PHONY: test-bench
test-bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...
	@echo "Benchmarks completed"

# Profiling targets
.PHONY: profile-cpu
profile-cpu:
	@echo "Running CPU profiling..."
	mkdir -p ./profiles
	$(GOTEST) -cpuprofile=./profiles/cpu.prof -bench=. ./...
	@echo "CPU profiling completed. Results in ./profiles/cpu.prof"
	@echo "View the profile with: go tool pprof ./profiles/cpu.prof"

.PHONY: profile-mem
profile-mem:
	@echo "Running memory profiling..."
	mkdir -p ./profiles
	$(GOTEST) -memprofile=./profiles/mem.prof -bench=. ./...
	@echo "Memory profiling completed. Results in ./profiles/mem.prof"
	@echo "View the profile with: go tool pprof ./profiles/mem.prof"

.PHONY: profile-block
profile-block:
	@echo "Running block profiling..."
	mkdir -p ./profiles
	$(GOTEST) -blockprofile=./profiles/block.prof -bench=. ./...
	@echo "Block profiling completed. Results in ./profiles/block.prof"
	@echo "View the profile with: go tool pprof ./profiles/block.prof"

.PHONY: profile-mutex
profile-mutex:
	@echo "Running mutex profiling..."
	mkdir -p ./profiles
	$(GOTEST) -mutexprofile=./profiles/mutex.prof -bench=. ./...
	@echo "Mutex profiling completed. Results in ./profiles/mutex.prof"
	@echo "View the profile with: go tool pprof ./profiles/mutex.prof"

.PHONY: profile-trace
profile-trace:
	@echo "Running execution tracing..."
	mkdir -p ./profiles
	$(GOTEST) -trace=./profiles/trace.out -bench=. ./...
	@echo "Execution tracing completed. Results in ./profiles/trace.out"
	@echo "View the trace with: go tool trace ./profiles/trace.out"

.PHONY: profile-all
profile-all: profile-cpu profile-mem profile-block profile-mutex profile-trace
	@echo "All profiling completed"

# Run tests with timeout
.PHONY: test-timeout
test-timeout:
	@echo "Running tests with 30s timeout..."
	$(GOTEST) -timeout 30s ./...
	@echo "Timeout tests completed"

# Run specific timeout behavior tests
.PHONY: test-timeout-behavior
test-timeout-behavior:
	@echo "Running timeout behavior tests..."
	$(GOTEST) -v ./... -run "Test.*Timeout"
	@echo "Timeout behavior tests completed"

# Run tests for a specific package
.PHONY: test-package
test-package:
	@echo "Running tests for a specific package..."
	@echo "Usage: make test-package PKG=./path/to/package"
	@if [ "$(PKG)" = "" ]; then \
		echo "Error: PKG is required. Example: make test-package PKG=./internal/domain"; \
		exit 1; \
	fi
	$(GOTEST) -v $(PKG)
	@echo "Package tests completed"

# Run tests matching a specific pattern
.PHONY: test-run
test-run:
	@echo "Running tests matching a specific pattern..."
	@echo "Usage: make test-run PATTERN=TestName"
	@if [ "$(PATTERN)" = "" ]; then \
		echo "Error: PATTERN is required. Example: make test-run PATTERN=TestCreateParent"; \
		exit 1; \
	fi
	$(GOTEST) -v ./... -run $(PATTERN)
	@echo "Pattern tests completed"

# Run tests with coverage matching a specific pattern
.PHONY: test-run-coverage
test-run-coverage:
	@echo "Running tests with coverage matching a specific pattern..."
	@echo "Usage: make test-run-coverage PATTERN=TestName"
	@if [ "$(PATTERN)" = "" ]; then \
		echo "Error: PATTERN is required. Example: make test-run-coverage PATTERN=TestCreateParent"; \
		exit 1; \
	fi
	mkdir -p ./coverage
	$(GOTEST) -v -coverprofile=./coverage/pattern.out ./... -run $(PATTERN)
	$(GOCMD) tool cover -html=./coverage/pattern.out -o ./coverage/pattern.html
	@echo "Pattern coverage report generated: ./coverage/pattern.html"

# Run tests with coverage for a specific package
.PHONY: test-package-coverage
test-package-coverage:
	@echo "Running tests with coverage for a specific package..."
	@echo "Usage: make test-package-coverage PKG=./path/to/package"
	@if [ "$(PKG)" = "" ]; then \
		echo "Error: PKG is required. Example: make test-package-coverage PKG=./internal/domain"; \
		exit 1; \
	fi
	mkdir -p ./coverage
	$(GOTEST) -v -cover -coverprofile=./coverage/$(shell basename $(PKG)).out $(PKG)
	$(GOCMD) tool cover -html=./coverage/$(shell basename $(PKG)).out -o ./coverage/$(shell basename $(PKG)).html
	@echo "Package coverage report generated: ./coverage/$(shell basename $(PKG)).html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf bin
	rm -rf coverage
	rm -f $(BINARY_NAME)
	@echo "Clean completed"

# Run the application locally
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) $(MAIN_PATH)

# Run the application with hot reloading for development
.PHONY: dev
dev: airconfig
	@echo "Running $(BINARY_NAME) in development mode with hot reloading..."
	air -c .air.toml

# Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	$(GOLINT) run
	@echo "Linting completed"

# Pre-commit checks
.PHONY: pre-commit
pre-commit: tidy fmt lint test vuln-check
	@echo "All pre-commit checks passed!"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	@echo "Formatting completed"

# Deploy the application
.PHONY: deploy
deploy: vuln-check
	@echo "Deploying $(BINARY_NAME)..."
	@echo "This is a placeholder. Implement actual deployment logic here."
	@echo "Deployment completed"

# Build Docker image
.PHONY: docker
docker: dockerfile
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built successfully"

# Run Docker container
.PHONY: docker-run
docker-run: docker
	@echo "Running Docker container $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "Docker container started"

# Docker Compose targets
.PHONY: docker-compose-up
docker-compose-up:
	@echo "Starting all services with Docker Compose..."
	docker-compose up -d
	@echo "All services started"

.PHONY: docker-compose-down
docker-compose-down:
	@echo "Stopping all services with Docker Compose..."
	docker-compose down
	@echo "All services stopped"

.PHONY: docker-compose-logs
docker-compose-logs:
	@echo "Showing logs for all services..."
	docker-compose logs -f

.PHONY: docker-compose-ps
docker-compose-ps:
	@echo "Showing status of all services..."
	docker-compose ps

.PHONY: docker-compose-restart
docker-compose-restart:
	@echo "Restarting all services..."
	docker-compose restart
	@echo "All services restarted"

.PHONY: docker-compose-build
docker-compose-build:
	@echo "Building all services..."
	docker-compose build
	@echo "All services built"

# Tidy and verify Go modules
.PHONY: tidy
tidy:
	@echo "Tidying Go modules..."
	$(GOMOD) tidy
	$(GOMOD) verify
	@echo "Go modules tidied and verified"

# Dependency management targets
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"

.PHONY: deps-upgrade
deps-upgrade:
	@echo "Upgrading dependencies..."
	go get -u ./...
	$(GOMOD) tidy
	@echo "Dependencies upgraded"

.PHONY: deps-graph
deps-graph:
	@echo "Generating dependency graph..."
	go install github.com/kisielk/godepgraph@latest
	mkdir -p ./docs/deps
	godepgraph -s github.com/abitofhelp/family-service2 | dot -Tpng -o ./docs/deps/dependency-graph.png
	@echo "Dependency graph generated at ./docs/deps/dependency-graph.png"

# Check for vulnerabilities in dependencies
.PHONY: vuln-check
vuln-check:
	@echo "Checking for vulnerabilities in dependencies..."
	$(GOVULNCHECK) ./...
	@echo "Vulnerability check completed"

# Verify translations
.PHONY: verify-translations
verify-translations:
	@echo "Verifying translations..."
	cd cmd/verify_translations && go run main.go
	@echo "Translation verification completed"

# Documentation targets
.PHONY: docs
docs:
	@echo "Generating documentation..."
	mkdir -p ./docs/godoc
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"
	@echo "Visit http://localhost:6060/pkg/github.com/abitofhelp/family-service2/ to view the documentation"

.PHONY: docs-pkgsite
docs-pkgsite:
	@echo "Generating documentation with pkgsite..."
	go install golang.org/x/pkgsite/cmd/pkgsite@latest
	pkgsite -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

.PHONY: docs-static
docs-static:
	@echo "Generating static documentation..."
	mkdir -p ./docs/godoc
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -url=/pkg/github.com/abitofhelp/family-service2/ > ./docs/godoc/index.html
	@echo "Static documentation generated at ./docs/godoc/index.html"

# Run MongoDB migrations
.PHONY: migrate-mongo
migrate-mongo:
	@echo "Running MongoDB migrations..."
	$(MIGRATE_MONGO)
	@echo "MongoDB migrations completed"

# Run PostgreSQL migrations
.PHONY: migrate-postgres
migrate-postgres:
	@echo "Running PostgreSQL migrations..."
	$(MIGRATE_POSTGRES)
	@echo "PostgreSQL migrations completed"

# Run all migrations
.PHONY: migrate
migrate: migrate-mongo migrate-postgres
	@echo "All migrations completed"

# Drop test databases
.PHONY: drop-test-dbs
drop-test-dbs:
	@echo "Dropping test databases..."
	chmod +x ./scripts/drop_test_databases.sh
	./scripts/drop_test_databases.sh
	@echo "Test databases dropped successfully"

# Recreate migration scripts
.PHONY: recreate-migrations
recreate-migrations:
	@echo "Recreating migration scripts..."
	chmod +x ./scripts/recreate_migrations.sh
	./scripts/recreate_migrations.sh
	@echo "Migration scripts recreated successfully"

# Recreate all integration tests and migration scripts
.PHONY: recreate-all
recreate-all: drop-test-dbs recreate-migrations migrate recreate-integration-tests
	@echo "All integration tests and migration scripts recreated successfully"

# Create a basic Dockerfile if one doesn't exist
.PHONY: dockerfile
dockerfile:
	@echo "Creating Dockerfile..."
	@if [ -f Dockerfile ]; then \
		echo "Dockerfile already exists. Skipping."; \
	else \
		echo "FROM golang:1.24-alpine AS builder" > Dockerfile; \
		echo "WORKDIR /app" >> Dockerfile; \
		echo "COPY . ." >> Dockerfile; \
		echo "RUN go mod download" >> Dockerfile; \
		echo "RUN go build -o family-service ./cmd/server" >> Dockerfile; \
		echo "" >> Dockerfile; \
		echo "FROM alpine:latest" >> Dockerfile; \
		echo "RUN apk --no-cache add ca-certificates" >> Dockerfile; \
		echo "WORKDIR /root/" >> Dockerfile; \
		echo "COPY --from=builder /app/family-service ." >> Dockerfile; \
		echo "COPY --from=builder /app/config ./config" >> Dockerfile; \
		echo "EXPOSE 8080" >> Dockerfile; \
		echo "CMD [\"./family-service\"]" >> Dockerfile; \
		echo "Dockerfile created successfully."; \
	fi

# Create a basic .air.toml configuration file if one doesn't exist
.PHONY: airconfig
airconfig:
	@echo "Creating .air.toml configuration file..."
	@if [ -f .air.toml ]; then \
		echo ".air.toml already exists. Skipping."; \
	else \
		echo "# .air.toml configuration file" > .air.toml; \
		echo "root = \"./\"" >> .air.toml; \
		echo "tmp_dir = \"tmp\"" >> .air.toml; \
		echo "" >> .air.toml; \
		echo "[build]" >> .air.toml; \
		echo "  cmd = \"go build -o ./tmp/main ./cmd/server\"" >> .air.toml; \
		echo "  bin = \"./tmp/main\"" >> .air.toml; \
		echo "  delay = 1000" >> .air.toml; \
		echo "  exclude_dir = [\"assets\", \"tmp\", \"vendor\", \"bin\"]" >> .air.toml; \
		echo "  include_ext = [\"go\", \"yaml\", \"yml\", \"graphql\"]" >> .air.toml; \
		echo "  exclude_regex = [\"_test\\.go\"]" >> .air.toml; \
		echo "" >> .air.toml; \
		echo "[log]" >> .air.toml; \
		echo "  time = true" >> .air.toml; \
		echo "" >> .air.toml; \
		echo "[color]" >> .air.toml; \
		echo "  main = \"magenta\"" >> .air.toml; \
		echo "  watcher = \"cyan\"" >> .air.toml; \
		echo "  build = \"yellow\"" >> .air.toml; \
		echo "  runner = \"green\"" >> .air.toml; \
		echo "" >> .air.toml; \
		echo "[screen]" >> .air.toml; \
		echo "  clear_on_rebuild = true" >> .air.toml; \
		echo ".air.toml created successfully."; \
	fi
