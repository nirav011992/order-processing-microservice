# Order Processing Microservice - Verification Guide

This guide walks through the complete process of running and verifying the order processing microservice.

## Prerequisites

1. **Docker Desktop** must be running
2. **Ports 5432, 8080, 9080, 9092** should be available
3. **Git Bash or Terminal** for running commands

## Step 1: Start Docker Desktop

Make sure Docker Desktop is running on your system. You should see the Docker icon in your system tray.

## Step 2: Verify System Requirements

```bash
# Check Docker version
docker --version
# Expected: Docker version 20.10.8 or higher

# Check Docker Compose version  
docker-compose --version
# Expected: docker-compose version 1.29.2 or higher

# Verify Docker is running
docker ps
# Expected: Should show running containers (if any) without errors
```

## Step 3: Navigate to Project Directory

```bash
cd /path/to/order-processing-microservice
```

## Step 4: Start the System

### Option A: Quick Start (Recommended)
```bash
./scripts/quick-start.sh
```

This script will:
1. Clean up any existing containers
2. Start infrastructure services (PostgreSQL, Kafka)
3. Wait for services to be ready
4. Start application services
5. Verify health checks
6. Provide API examples

### Option B: Manual Start
```bash
# Clean up any existing containers
docker-compose down --volumes --remove-orphans

# Start infrastructure first
docker-compose up -d postgres zookeeper kafka

# Wait for infrastructure to be ready (30-45 seconds)
sleep 45

# Start application services
docker-compose up -d producer-api consumer status-api

# Check all services are running
docker-compose ps
```

## Step 5: Verify Services Are Running

### Check Container Status
```bash
docker-compose ps
```

Expected output:
```
        Name                      Command                  State                    Ports
------------------------------------------------------------------------------------------------
order-consumer         /app/consumer                    Up
order-kafka            /etc/confluent/docker/run        Up      0.0.0.0:29092->29092/tcp,
                                                                 0.0.0.0:9092->9092/tcp
order-postgres         docker-entrypoint.sh postgres   Up      0.0.0.0:5432->5432/tcp
order-producer-api     /app/producer                    Up      0.0.0.0:8080->8080/tcp
order-status-api       /app/status-api                  Up      0.0.0.0:9080->9080/tcp
order-zookeeper        /etc/confluent/docker/run        Up      2181/tcp
```

### Health Check Endpoints
```bash
# Status API Health Check
curl http://localhost:9080/health

# Expected Response:
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "order-processing-microservice",
  "version": "1.0.0"
}

# Producer API Health Check (if health endpoint available)
curl http://localhost:8080/health
```

## Step 6: Test the APIs

### Test 1: Check Initial Statistics
```bash
curl http://localhost:9080/api/v1/status/stats
```

Expected response:
```json
{
  "data": {
    "total": 0,
    "pending": 0,
    "processing": 0,
    "completed": 0,
    "canceled": 0,
    "failed": 0
  }
}
```

### Test 2: Create an Order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "items": [
      {
        "product_id": "456e7890-e89b-12d3-a456-426614174001",
        "quantity": 2,
        "price": 29.99
      }
    ]
  }'
```

Expected response:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "pending",
    "items": [
      {
        "product_id": "456e7890-e89b-12d3-a456-426614174001",
        "quantity": 2,
        "price": 29.99,
        "total": 59.98
      }
    ],
    "total_amount": 59.98,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "message": "Order created successfully"
}
```

### Test 3: Get Order Details
```bash
# Replace {order_id} with the ID from the previous response
curl http://localhost:8080/api/v1/orders/{order_id}
```

### Test 4: Wait for Processing (10-15 seconds)
The consumer service will automatically process the order:
- Status changes: pending → processing → completed (or failed)

### Test 5: Check Updated Statistics
```bash
curl http://localhost:9080/api/v1/status/stats
```

You should now see updated counts reflecting the created and processed orders.

### Test 6: Get Orders by Status
```bash
# Get completed orders
curl http://localhost:9080/api/v1/status/orders/completed

# Get pending orders
curl http://localhost:9080/api/v1/status/orders/pending

# Get processing orders  
curl http://localhost:9080/api/v1/status/orders/processing
```

### Test 7: Get Orders by Customer
```bash
curl http://localhost:8080/api/v1/customers/123e4567-e89b-12d3-a456-426614174000/orders
```

## Step 7: Run Automated Tests

```bash
# Run the comprehensive API test suite
./scripts/test-api.sh
```

This script will:
1. Check service availability
2. Create multiple test orders
3. Verify order processing
4. Test all API endpoints
5. Display results with colored output

## Step 8: Monitor the System

### View Logs
```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f producer-api
docker-compose logs -f consumer
docker-compose logs -f status-api
```

### Check Database
```bash
# Connect to PostgreSQL
docker exec -it order-postgres psql -U postgres -d orders_db

# View orders
SELECT id, customer_id, status, total_amount, created_at FROM orders;

# View order items
SELECT oi.*, o.status FROM order_items oi 
JOIN orders o ON oi.order_id = o.id;
```

### Check Kafka Topics
```bash
# List Kafka topics
docker exec -it order-kafka kafka-topics --list --bootstrap-server localhost:9092

# View messages in the order-events topic
docker exec -it order-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic order-events \
  --from-beginning
```

## Expected Behavior

1. **Order Creation**: Creates order with "pending" status
2. **Event Publishing**: Publishes "order.created" event to Kafka
3. **Consumer Processing**: Consumer picks up event and processes order
4. **Status Updates**: Order status changes to "processing" then "completed" (90% success rate)
5. **Event Chain**: Each status change publishes corresponding events
6. **Persistence**: All changes are saved to PostgreSQL with optimistic locking

## Troubleshooting

### Services Not Starting
```bash
# Check Docker Desktop is running
docker version

# Check port availability
netstat -tulpn | grep :8080
netstat -tulpn | grep :9080
netstat -tulpn | grep :5432

# Restart services
docker-compose restart
```

### API Returning Errors
```bash
# Check service logs
docker-compose logs producer-api
docker-compose logs consumer

# Verify database connection
docker-compose logs postgres
```

### Database Issues
```bash
# Check if tables were created
docker exec -it order-postgres psql -U postgres -d orders_db -c "\\dt"

# Manually create tables if needed
docker exec -it order-postgres psql -U postgres -d orders_db -f /path/to/schema.sql
```

### Kafka Issues
```bash
# Check Kafka broker
docker-compose logs kafka

# Verify topic creation
docker exec -it order-kafka kafka-topics --list --bootstrap-server localhost:9092
```

## Clean Up

```bash
# Stop all services
docker-compose down

# Remove volumes (this will delete all data)
docker-compose down --volumes

# Remove everything including orphaned containers
docker-compose down --volumes --remove-orphans
```

## Success Indicators

✅ All containers are running (`docker-compose ps`)  
✅ Health checks return 200 OK  
✅ Order creation returns 201 Created  
✅ Orders are processed (status changes from pending to completed)  
✅ Statistics show accurate counts  
✅ Logs show successful processing  
✅ Database contains order data  
✅ Kafka topics contain events  

## Performance Notes

- **Startup Time**: Allow 1-2 minutes for full system startup
- **Processing Time**: Orders typically process within 3-10 seconds
- **Success Rate**: ~90% of orders complete successfully (10% fail for demo purposes)
- **Throughput**: System can handle multiple concurrent orders

This microservice demonstrates enterprise-grade architecture patterns and is ready for development, testing, and production deployment.