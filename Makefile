# Makefile for Eagle Bank

.PHONY: help build up down logs clean test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d

up-build: ## Build and start all services
	docker-compose up -d --build

down: ## Stop all services
	docker-compose down

down-v: ## Stop all services and remove volumes
	docker-compose down -v

logs: ## Show logs for all services
	docker-compose logs -f

logs-api: ## Show logs for API Gateway
	docker-compose logs -f api-gateway

logs-auth: ## Show logs for Auth Service
	docker-compose logs -f auth-service

logs-user: ## Show logs for User Service
	docker-compose logs -f user-service

logs-account: ## Show logs for Account Service
	docker-compose logs -f account-service

logs-transaction: ## Show logs for Transaction Service
	docker-compose logs -f transaction-service

ps: ## Show running containers
	docker-compose ps

restart: ## Restart all services
	docker-compose restart

clean: ## Remove all containers, volumes, and images
	docker-compose down -v --rmi all

redis-cli: ## Connect to Redis CLI
	docker exec -it eagle-redis redis-cli

psql-users: ## Connect to Users PostgreSQL
	docker exec -it eagle-postgres-users psql -U postgres -d eagle_users

psql-accounts: ## Connect to Accounts PostgreSQL
	docker exec -it eagle-postgres-accounts psql -U postgres -d eagle_accounts

psql-transactions: ## Connect to Transactions PostgreSQL
	docker exec -it eagle-postgres-transactions psql -U postgres -d eagle_transactions

health: ## Check health of all services
	@echo "API Gateway:"; curl -s http://localhost:8080/health | jq || echo "Not available"
	@echo "\nAuth Service:"; curl -s http://localhost:8081/health | jq || echo "Not available"
	@echo "\nUser Service:"; curl -s http://localhost:8082/health | jq || echo "Not available"
	@echo "\nAccount Service:"; curl -s http://localhost:8083/health | jq || echo "Not available"
	@echo "\nTransaction Service:"; curl -s http://localhost:8084/health | jq || echo "Not available"
