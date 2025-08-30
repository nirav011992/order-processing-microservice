# API Documentation

This document provides comprehensive API documentation for the Order Processing Microservice.

## Base URLs

- **Producer API**: `http://localhost:8080`
- **Status API**: `http://localhost:9080`

## Authentication

Currently, the API does not require authentication. In a production environment, you should implement proper authentication and authorization mechanisms.

## Producer API Endpoints

### Health Check

Check the health status of the Producer API service.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "service": "order-processing-microservice",
  "status": "healthy",
  "timestamp": "2025-08-30T12:00:00Z",
  "version": "1.0.0"
}
```

**Status Codes:**
- `200 OK` - Service is healthy
- `503 Service Unavailable` - Service is unhealthy

### Create Order

Create a new order in the system.

**Endpoint:** `POST /api/v1/orders`

**Request Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "customer_id": "123e4567-e89b-12d3-a456-426614174000",
  "items": [
    {
      "product_id": "987fcdeb-51a2-43d4-b123-456789abcdef",
      "name": "Product Name",
      "price": 29.99,
      "quantity": 2
    }
  ],
  "total_amount": 59.98
}
```

**Request Body Fields:**
- `customer_id` (string, required): UUID of the customer placing the order
- `items` (array, required): Array of order items
  - `product_id` (string, required): UUID of the product
  - `name` (string, required): Name of the product
  - `price` (number, required): Unit price of the product (must be > 0)
  - `quantity` (integer, required): Quantity ordered (must be > 0)
- `total_amount` (number, optional): Total amount of the order (calculated if not provided)

**Response:**
```json
{
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "pending",
    "items": [
      {
        "id": "c9bf9e57-1685-4c89-bafb-ff5af830be8a",
        "order_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "product_id": "987fcdeb-51a2-43d4-b123-456789abcdef",
        "quantity": 2,
        "price": 29.99,
        "total": 59.98
      }
    ],
    "total_amount": 59.98,
    "created_at": "2025-08-30T12:00:00Z",
    "updated_at": "2025-08-30T12:00:00Z"
  },
  "message": "Order created successfully"
}
```

**Status Codes:**
- `201 Created` - Order created successfully
- `400 Bad Request` - Invalid request body or validation errors
- `500 Internal Server Error` - Server error

**Validation Rules:**
- Customer ID must be a valid UUID
- At least one item is required
- Each item must have a valid product ID (UUID)
- Price must be greater than 0
- Quantity must be greater than 0

### Get Order

Retrieve a specific order by its ID.

**Endpoint:** `GET /api/v1/orders/{order_id}`

**Path Parameters:**
- `order_id` (string, required): UUID of the order

**Response:**
```json
{
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "completed",
    "items": [
      {
        "id": "c9bf9e57-1685-4c89-bafb-ff5af830be8a",
        "order_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "product_id": "987fcdeb-51a2-43d4-b123-456789abcdef",
        "quantity": 2,
        "price": 29.99,
        "total": 59.98
      }
    ],
    "total_amount": 59.98,
    "created_at": "2025-08-30T12:00:00Z",
    "updated_at": "2025-08-30T12:00:30Z"
  }
}
```

**Status Codes:**
- `200 OK` - Order retrieved successfully
- `400 Bad Request` - Invalid order ID format
- `404 Not Found` - Order not found
- `500 Internal Server Error` - Server error

### Get Customer Orders

Retrieve all orders for a specific customer with pagination support.

**Endpoint:** `GET /api/v1/orders/customer/{customer_id}`

**Path Parameters:**
- `customer_id` (string, required): UUID of the customer

**Query Parameters:**
- `limit` (integer, optional): Maximum number of orders to return (default: 10, max: 100)
- `offset` (integer, optional): Number of orders to skip for pagination (default: 0)

**Response:**
```json
{
  "data": {
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "orders": [
      {
        "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "customer_id": "123e4567-e89b-12d3-a456-426614174000",
        "status": "completed",
        "total_amount": 59.98,
        "created_at": "2025-08-30T12:00:00Z",
        "updated_at": "2025-08-30T12:00:30Z"
      }
    ],
    "meta": {
      "limit": 10,
      "offset": 0,
      "count": 1
    }
  }
}
```

**Status Codes:**
- `200 OK` - Orders retrieved successfully
- `400 Bad Request` - Invalid customer ID or query parameters
- `500 Internal Server Error` - Server error

## Status API Endpoints

### Health Check

Check the health status of the Status API service.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "service": "order-processing-microservice",
  "status": "healthy",
  "timestamp": "2025-08-30T12:00:00Z",
  "version": "1.0.0"
}
```

### Get Order Statistics

Retrieve comprehensive order statistics.

**Endpoint:** `GET /api/v1/status/stats`

**Response:**
```json
{
  "data": {
    "pending": 5,
    "processing": 3,
    "completed": 42,
    "failed": 2,
    "canceled": 1,
    "total": 53
  }
}
```

**Status Codes:**
- `200 OK` - Statistics retrieved successfully
- `500 Internal Server Error` - Server error

### Get Orders by Status

Retrieve orders filtered by their status with pagination support.

**Endpoint:** `GET /api/v1/status/orders/{status}`

**Path Parameters:**
- `status` (string, required): Order status (`pending`, `processing`, `completed`, `failed`, `canceled`)

**Query Parameters:**
- `limit` (integer, optional): Maximum number of orders to return (default: 10, max: 100)
- `offset` (integer, optional): Number of orders to skip for pagination (default: 0)

**Response:**
```json
{
  "data": {
    "meta": {
      "status": "completed",
      "limit": 10,
      "offset": 0,
      "count": 2
    },
    "orders": [
      {
        "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "customer_id": "123e4567-e89b-12d3-a456-426614174000",
        "status": "completed",
        "items": [
          {
            "id": "c9bf9e57-1685-4c89-bafb-ff5af830be8a",
            "order_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
            "product_id": "987fcdeb-51a2-43d4-b123-456789abcdef",
            "quantity": 2,
            "price": 29.99,
            "total": 59.98
          }
        ],
        "total_amount": 59.98,
        "created_at": "2025-08-30T12:00:00Z",
        "updated_at": "2025-08-30T12:00:30Z"
      }
    ]
  }
}
```

**Status Codes:**
- `200 OK` - Orders retrieved successfully
- `400 Bad Request` - Invalid status or query parameters
- `500 Internal Server Error` - Server error

### Get System Metrics

Retrieve comprehensive system metrics including order statistics and system information.

**Endpoint:** `GET /api/v1/status/metrics`

**Response:**
```json
{
  "data": {
    "orders": {
      "pending": 5,
      "processing": 3,
      "completed": 42,
      "failed": 2,
      "canceled": 1,
      "total": 53
    },
    "system": {
      "timestamp": "2025-08-30T12:00:00Z",
      "uptime": "1h23m45s"
    }
  }
}
```

**Status Codes:**
- `200 OK` - Metrics retrieved successfully
- `500 Internal Server Error` - Server error

## Order Status Lifecycle

Orders progress through the following statuses:

1. **pending** - Initial state when order is created
2. **processing** - Order is being processed by the system
3. **completed** - Order has been processed successfully
4. **failed** - Order processing failed
5. **canceled** - Order has been canceled

## Error Response Format

All API endpoints return errors in the following standardized format:

```json
{
  "error": "Error title",
  "message": "Detailed error message",
  "code": 400,
  "timestamp": "2025-08-30T12:00:00Z"
}
```

## Common Error Codes

- `400 Bad Request` - Invalid request parameters, validation errors
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server-side error
- `503 Service Unavailable` - Service temporarily unavailable

## Rate Limiting

Currently, there are no rate limits implemented. In a production environment, consider implementing rate limiting to prevent abuse.

## Pagination

APIs that return multiple items support pagination using `limit` and `offset` parameters:

- `limit`: Maximum number of items to return (default: 10, max: 100)
- `offset`: Number of items to skip (default: 0)

The response includes a `meta` object with pagination information:

```json
{
  "meta": {
    "limit": 10,
    "offset": 0,
    "count": 5
  }
}
```

## Example Usage

### Creating and Tracking an Order

1. **Create an order:**
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "items": [
      {
        "product_id": "987fcdeb-51a2-43d4-b123-456789abcdef",
        "name": "Test Product",
        "price": 29.99,
        "quantity": 2
      }
    ]
  }'
```

2. **Check order status:**
```bash
curl http://localhost:8080/api/v1/orders/{order_id}
```

3. **Monitor order statistics:**
```bash
curl http://localhost:9080/api/v1/status/stats
```

4. **Get completed orders:**
```bash
curl http://localhost:9080/api/v1/status/orders/completed
```

## Webhook Support

Currently, webhooks are not supported. Order status changes are communicated through Kafka events internally. Consider implementing webhooks for external systems integration in future versions.

## API Versioning

The API currently uses URL path versioning (`/api/v1/`). Future versions will increment the version number (`/api/v2/`, etc.) while maintaining backward compatibility.

## OpenAPI Specification

For a machine-readable API specification, refer to the OpenAPI 3.0 specification file at `/api/openapi/swagger.yaml` (to be implemented).

## SDK and Client Libraries

Currently, no official SDKs are provided. Consider implementing client libraries for popular programming languages based on the OpenAPI specification.