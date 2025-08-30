#!/bin/bash

echo "=== Order Processing Microservice API Tests ==="
echo ""

# Configuration
PRODUCER_URL="http://localhost:8080"
STATUS_URL="http://localhost:9080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function to check if service is running
check_service() {
    local url=$1
    local name=$2
    echo -n "Checking $name..."
    if curl -s -f "$url/health" > /dev/null 2>&1; then
        echo -e " ${GREEN}✅ Running${NC}"
        return 0
    else
        echo -e " ${RED}❌ Not available${NC}"
        return 1
    fi
}

# Helper function to make API call
api_call() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo ""
    echo -e "${YELLOW}Testing: $description${NC}"
    echo "Method: $method"
    echo "URL: $url"
    
    if [ -n "$data" ]; then
        echo "Data: $data"
        echo ""
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$url")
    else
        echo ""
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X "$method" "$url")
    fi
    
    # Extract HTTP status code
    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    body=$(echo "$response" | sed '$d')
    
    echo "Response (HTTP $http_status):"
    echo "$body" | jq . 2>/dev/null || echo "$body"
    
    if [ "$http_status" -ge 200 ] && [ "$http_status" -lt 300 ]; then
        echo -e "${GREEN}✅ Success${NC}"
        
        # Extract order ID if this is a create order response
        if [[ "$description" == *"Create Order"* ]]; then
            ORDER_ID=$(echo "$body" | jq -r '.data.id' 2>/dev/null)
            if [ "$ORDER_ID" != "null" ] && [ -n "$ORDER_ID" ]; then
                echo "Created Order ID: $ORDER_ID"
            fi
        fi
    else
        echo -e "${RED}❌ Failed${NC}"
    fi
}

echo "Checking service availability..."
check_service "$STATUS_URL" "Status API"
check_service "$PRODUCER_URL" "Producer API"

if ! check_service "$PRODUCER_URL" "Producer API" || ! check_service "$STATUS_URL" "Status API"; then
    echo ""
    echo -e "${YELLOW}⚠️  Some services are not available. Make sure to start them with:${NC}"
    echo "   docker-compose up -d"
    echo ""
    echo "Continuing with available services..."
fi

# Test 1: Health Check
api_call "GET" "$STATUS_URL/health" "" "Health Check"

# Test 2: Get Initial Stats
api_call "GET" "$STATUS_URL/api/v1/status/stats" "" "Get Order Statistics"

# Test 3: Create Order
echo ""
echo -e "${YELLOW}🛍️  Creating sample orders...${NC}"

# Sample customer and product UUIDs
CUSTOMER_ID="123e4567-e89b-12d3-a456-426614174000"
PRODUCT_ID_1="456e7890-e89b-12d3-a456-426614174001"
PRODUCT_ID_2="789e0123-e89b-12d3-a456-426614174002"

ORDER_DATA='{
  "customer_id": "'$CUSTOMER_ID'",
  "items": [
    {
      "product_id": "'$PRODUCT_ID_1'",
      "quantity": 2,
      "price": 29.99
    },
    {
      "product_id": "'$PRODUCT_ID_2'",
      "quantity": 1,
      "price": 15.50
    }
  ]
}'

api_call "POST" "$PRODUCER_URL/api/v1/orders" "$ORDER_DATA" "Create Order #1"

# Test 4: Create another order
ORDER_DATA_2='{
  "customer_id": "'$CUSTOMER_ID'",
  "items": [
    {
      "product_id": "'$PRODUCT_ID_1'",
      "quantity": 1,
      "price": 29.99
    }
  ]
}'

api_call "POST" "$PRODUCER_URL/api/v1/orders" "$ORDER_DATA_2" "Create Order #2"

# Wait a bit for processing
echo ""
echo -e "${YELLOW}⏳ Waiting 3 seconds for order processing...${NC}"
sleep 3

# Test 5: Get Orders by Customer
api_call "GET" "$PRODUCER_URL/api/v1/customers/$CUSTOMER_ID/orders" "" "Get Orders by Customer"

# Test 6: Get Orders by Status
api_call "GET" "$STATUS_URL/api/v1/status/orders/pending" "" "Get Pending Orders"
api_call "GET" "$STATUS_URL/api/v1/status/orders/processing" "" "Get Processing Orders"
api_call "GET" "$STATUS_URL/api/v1/status/orders/completed" "" "Get Completed Orders"

# Test 7: Get Updated Stats
api_call "GET" "$STATUS_URL/api/v1/status/stats" "" "Get Updated Order Statistics"

# Test 8: Get System Metrics
api_call "GET" "$STATUS_URL/api/v1/status/metrics" "" "Get System Metrics"

echo ""
echo "=== API Testing Complete ==="
echo ""
echo -e "${GREEN}🎉 All tests completed!${NC}"
echo ""
echo "💡 Tips:"
echo "   • Monitor logs: docker-compose logs -f"
echo "   • Check Kafka topics: docker exec -it order-kafka kafka-topics --list --bootstrap-server localhost:9092"
echo "   • Access PostgreSQL: docker exec -it order-postgres psql -U postgres -d orders_db"
echo ""
echo -e "${YELLOW}📊 To view more detailed order information:${NC}"
echo "   curl $STATUS_URL/api/v1/status/orders/completed | jq"