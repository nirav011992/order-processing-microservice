#!/bin/bash

echo "🚀 Order Processing Microservice - Quick Start"
echo "============================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed or not in PATH${NC}"
    echo "Please install Docker and try again."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed or not in PATH${NC}"
    echo "Please install Docker Compose and try again."
    exit 1
fi

echo -e "${GREEN}✅ Docker and Docker Compose are available${NC}"
echo ""

# Function to wait for service to be healthy
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1
    
    echo -n "Waiting for $name to be ready"
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$url" > /dev/null 2>&1; then
            echo -e " ${GREEN}✅ Ready!${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo -e " ${RED}❌ Timeout waiting for $name${NC}"
    return 1
}

# Step 1: Clean up any existing containers
echo -e "${BLUE}📦 Cleaning up existing containers...${NC}"
docker-compose down --volumes --remove-orphans > /dev/null 2>&1

# Step 2: Start the infrastructure services first
echo -e "${BLUE}🏗️  Starting infrastructure services (PostgreSQL, Kafka)...${NC}"
docker-compose up -d postgres zookeeper kafka

echo ""
echo -e "${YELLOW}⏳ Waiting for infrastructure to be ready...${NC}"
sleep 15

# Step 3: Start the application services
echo -e "${BLUE}🚀 Starting application services...${NC}"
docker-compose up -d producer-api consumer status-api

echo ""
echo -e "${YELLOW}⏳ Waiting for services to start...${NC}"
sleep 10

# Step 4: Check service health
echo ""
echo -e "${BLUE}🔍 Checking service health...${NC}"

wait_for_service "http://localhost:9080/health" "Status API"
wait_for_service "http://localhost:8080/health" "Producer API" || true

# Step 5: Display service information
echo ""
echo -e "${GREEN}🎉 Order Processing Microservice is now running!${NC}"
echo ""
echo -e "${BLUE}📋 Service Endpoints:${NC}"
echo "   Producer API:  http://localhost:8080"
echo "   Status API:    http://localhost:9080"
echo "   Health Check:  http://localhost:9080/health"
echo ""
echo -e "${BLUE}🗄️  Infrastructure:${NC}"
echo "   PostgreSQL:    localhost:5432"
echo "   Kafka:         localhost:9092"
echo ""

# Step 6: Show sample API calls
echo -e "${BLUE}🧪 Sample API Calls:${NC}"
echo ""
echo "1. Check system health:"
echo "   curl http://localhost:9080/health"
echo ""
echo "2. Create an order:"
echo '   curl -X POST http://localhost:8080/api/v1/orders \'
echo '     -H "Content-Type: application/json" \'
echo '     -d '"'"'{'
echo '       "customer_id": "123e4567-e89b-12d3-a456-426614174000",'
echo '       "items": [{'
echo '         "product_id": "456e7890-e89b-12d3-a456-426614174001",'
echo '         "quantity": 2,'
echo '         "price": 29.99'
echo '       }]'
echo '     }'"'"
echo ""
echo "3. Get order statistics:"
echo "   curl http://localhost:9080/api/v1/status/stats"
echo ""
echo "4. Get orders by status:"
echo "   curl http://localhost:9080/api/v1/status/orders/completed"
echo ""

# Step 7: Offer to run tests
echo -e "${YELLOW}🧪 Would you like to run the API tests now? (y/n)${NC}"
read -r response

if [[ "$response" =~ ^[Yy]$ ]]; then
    echo ""
    echo -e "${BLUE}🏃 Running API tests...${NC}"
    ./scripts/test-api.sh
fi

echo ""
echo -e "${BLUE}📖 Useful Commands:${NC}"
echo "   View logs:           docker-compose logs -f"
echo "   View specific logs:  docker-compose logs -f producer-api"
echo "   Stop services:       docker-compose down"
echo "   Run API tests:       ./scripts/test-api.sh"
echo "   View project help:   make help"
echo ""
echo -e "${GREEN}✨ Happy coding!${NC}"