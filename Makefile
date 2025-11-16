.PHONY: help build test clean run-producer run-consumer docker-up docker-down docker-logs deps lint format

deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download

build:
	@echo "🔨 Building binaries..."
	go build -o bin/consumer ./cmd/consumer
	@echo "✅ Build complete"

test:
	@echo "🧪 Running tests..."
	go test -v ./internal/...
	@echo "✅ Tests complete"

test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

lint:
	@echo "🔍 Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run ./...
	@echo "✅ Linting complete"

format:
	@echo "💅 Formatting code..."
	go fmt ./...
	@command -v goimports >/dev/null 2>&1 || { echo "Installing goimports..."; go install golang.org/x/tools/cmd/goimports@latest; }
	goimports -w .
	@echo "✅ Formatting complete"

# Running commands
run-producer:
	@echo "🚀 Starting message producer..."
	go run cmd/producer/main.go

run-consumer:
	@echo "🔄 Starting message consumer..."
	go run cmd/consumer/main.go

docker-up:
	@echo "🐳 Starting Docker services..."
	cd deployments && docker compose up --build
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Docker services started"
	@echo "   RabbitMQ Management: http://localhost:15672 (admin/password)"
	@echo "   PostgreSQL: localhost:5432"

docker-stop:
	@echo "🛑 Stopping Docker services..."
	cd deployments && docker compose stop && docker compose down -v
	@echo "✅ Docker services stopped"

docker-reset:
	$(MAKE) docker-stop-all
	cd deployments && docker compose up -d
	@echo "✅ Docker services reset"

watch-consumer:
	@echo "👀 Watching consumer (auto-restart on changes)..."
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	air -c .air-consumer.toml

watch-producer:
	@echo "👀 Watching producer (auto-restart on changes)..."
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	air -c .air-producer.toml

docker-build:
	@echo "🏗️  Building Docker images..."
	docker build -t webhook-processor:latest .
	@echo "✅ Docker image built"

full-check: deps format lint test benchmark
	@echo "✅ Full check pipeline complete"

reset: clean docker-down docker-up
	@echo "✅ Full reset complete"
