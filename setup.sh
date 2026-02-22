#!/bin/bash

# Setup script for Eagle Bank

echo "🏦 Eagle Bank Setup Script"
echo "=========================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

echo "✅ Docker and Docker Compose are installed"
echo ""

# Stop any existing containers
echo "🛑 Stopping any existing containers..."
docker-compose down -v 2>/dev/null || true

echo ""
echo "🔨 Building Docker images..."
docker-compose build

echo ""
echo "🚀 Starting services..."
docker-compose up -d

echo ""
echo "⏳ Waiting for services to be healthy..."
sleep 10

# Check service health
echo ""
echo "🔍 Checking service health..."

check_health() {
    local service=$1
    local port=$2
    local response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$port/health 2>/dev/null)
    if [ "$response" = "200" ]; then
        echo "  ✅ $service is healthy"
        return 0
    else
        echo "  ⏳ $service is starting..."
        return 1
    fi
}

# Wait up to 60 seconds for all services
max_attempts=12
attempt=0

while [ $attempt -lt $max_attempts ]; do
    all_healthy=true
    
    check_health "API Gateway" 8080 || all_healthy=false
    check_health "Auth Service" 8081 || all_healthy=false
    check_health "User Service" 8082 || all_healthy=false
    check_health "Account Service" 8083 || all_healthy=false
    check_health "Transaction Service" 8084 || all_healthy=false
    
    if [ "$all_healthy" = true ]; then
        break
    fi
    
    attempt=$((attempt + 1))
    if [ $attempt -lt $max_attempts ]; then
        sleep 5
    fi
done

echo ""
if [ "$all_healthy" = true ]; then
    echo "✨ All services are running!"
    echo ""
    echo "📝 Service URLs:"
    echo "  API Gateway:          http://localhost:8080"
    echo "  Auth Service:         http://localhost:8081"
    echo "  User Service:         http://localhost:8082"
    echo "  Account Service:      http://localhost:8083"
    echo "  Transaction Service:  http://localhost:8084"
    echo ""
    echo "📚 Database Connections:"
    echo "  Users DB:        localhost:5432 (postgres/postgres)"
    echo "  Accounts DB:     localhost:5433 (postgres/postgres)"
    echo "  Transactions DB: localhost:5434 (postgres/postgres)"
    echo ""
    echo "💾 Redis: localhost:6379"
    echo ""
    echo "📖 See README.md for API documentation"
    echo ""
    echo "🧪 Run tests with: bash test-api.sh"
    echo "📊 View logs with: docker-compose logs -f"
    echo "🛑 Stop with: docker-compose down"
else
    echo "⚠️  Some services are still starting. Check logs with:"
    echo "  docker-compose logs -f"
fi
