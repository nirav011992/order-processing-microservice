.PHONY: help build run test clean docker-build docker-up docker-down

# Variables
PRODUCER_BINARY=bin/producer
CONSUMER_BINARY=bin/consumer
STATUS_API_BINARY=bin/status-api
CONFIG_FILE?=configs/local.env

# Help
help: ## Display this help screen
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build
build: ## Build all binaries
	@echo "Building producer..."
	@mkdir -p bin
	@go build -o $(PRODUCER_BINARY) ./cmd/producer
	@echo "Building consumer..."
	@go build -o $(CONSUMER_BINARY) ./cmd/consumer
	@echo "Building status API..."
	@go build -o $(STATUS_API_BINARY) ./cmd/status-api
	@echo "Build completed!"

# Run individual services
run-producer: build ## Run the producer API server
	@echo "Starting producer API server..."
	@./$(PRODUCER_BINARY) $(CONFIG_FILE)

run-consumer: build ## Run the order consumer
	@echo "Starting order consumer..."
	@./$(CONSUMER_BINARY) $(CONFIG_FILE)

run-status: build ## Run the status API server
	@echo "Starting status API server..."
	@./$(STATUS_API_BINARY) $(CONFIG_FILE)

# Test
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Code quality
lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	@go mod tidy

# Docker
docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker build -t order-producer -f Dockerfile.producer .
	@docker build -t order-consumer -f Dockerfile.consumer .
	@docker build -t order-status-api -f Dockerfile.status .

docker-up: ## Start services using Docker Compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

docker-down: ## Stop services using Docker Compose
	@echo "Stopping services with Docker Compose..."
	@docker-compose down

docker-logs: ## View Docker Compose logs
	@docker-compose logs -f

# Database
db-migrate: ## Run database migrations (create tables)
	@echo "Creating database tables..."
	@./$(PRODUCER_BINARY) $(CONFIG_FILE) -migrate

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@docker-compose down --volumes --remove-orphans

# Development helpers
dev-setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	@docker-compose up -d postgres kafka zookeeper
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Development environment ready!"

dev-teardown: ## Tear down development environment
	@echo "Tearing down development environment..."
	@docker-compose down --volumes

# Generate sample orders
generate-orders: build ## Generate sample orders for testing
	@echo "Generating sample orders..."
	@./scripts/create_test_orders.sh