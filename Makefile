.PHONY: help build run seed test test-race docker-up docker-down docker-logs clean

help: ## Display available commands
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Compile binary into bin/server
	go build -o bin/server ./cmd/api

run: ## Run application locally
	go run ./cmd/api

seed: ## Seed database with sample users, teams, and tasks
	go run ./cmd/seed

test: ## Run all unit tests
	go test -v ./...

test-race: ## Run all unit tests with race detector enabled
	CGO_ENABLED=1 go test -race -v ./...

docker-up: ## Start PostgreSQL, Redis, and API via Docker Compose
	docker compose up --build -d

docker-down: ## Stop all Docker Compose services
	docker compose down

docker-logs: ## View logs from Docker Compose
	docker compose logs -f

clean: ## Clean build artefacts and coverage
	rm -rf bin/ coverage.out
