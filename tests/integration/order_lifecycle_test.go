package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"order-processing-microservice/internal/models"
)

const (
	producerAPIURL = "http://localhost:8080"
	statusAPIURL   = "http://localhost:9080"
)

func TestOrderLifecycle_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	// Wait for services to be ready
	err := waitForService(producerAPIURL + "/health")
	require.NoError(t, err, "Producer API should be available")
	
	err = waitForService(statusAPIURL + "/health")
	require.NoError(t, err, "Status API should be available")

	t.Run("Complete Order Lifecycle", func(t *testing.T) {
		// Step 1: Create an order
		customerID := uuid.New()
		productID := uuid.New()
		
		createOrderReq := models.CreateOrderRequest{
			CustomerID: customerID,
			Items: []models.CreateOrderItemRequest{
				{
					ProductID: productID,
					Name:      "Integration Test Product",
					Price:     49.99,
					Quantity:  2,
				},
			},
		}
		
		orderResp, err := createOrder(createOrderReq)
		require.NoError(t, err, "Should create order successfully")
		require.NotNil(t, orderResp)
		assert.Equal(t, models.OrderStatusPending, orderResp.Data.Status)
		assert.Equal(t, customerID, orderResp.Data.CustomerID)
		assert.Equal(t, 99.98, orderResp.Data.TotalAmount)
		
		orderID := orderResp.Data.ID
		t.Logf("Created order: %s", orderID)
		
		// Step 2: Verify order can be retrieved
		order, err := getOrder(orderID)
		require.NoError(t, err, "Should retrieve order successfully")
		assert.Equal(t, orderID, order.Data.ID)
		assert.Equal(t, customerID, order.Data.CustomerID)
		
		// Step 3: Wait for order to be processed by consumer
		// The consumer should process the order and update status to completed
		t.Log("Waiting for order to be processed...")
		
		var processedOrder *models.GetOrderResponse
		maxWaitTime := 30 * time.Second
		checkInterval := 2 * time.Second
		
		for elapsed := time.Duration(0); elapsed < maxWaitTime; elapsed += checkInterval {
			time.Sleep(checkInterval)
			
			processedOrder, err = getOrder(orderID)
			require.NoError(t, err, "Should retrieve order successfully")
			
			if processedOrder.Data.Status == models.OrderStatusCompleted {
				t.Logf("Order processed successfully in %v", elapsed)
				break
			}
			
			t.Logf("Order status: %s, waiting...", processedOrder.Data.Status)
		}
		
		// Verify final state
		assert.Equal(t, models.OrderStatusCompleted, processedOrder.Data.Status, "Order should be completed")
		assert.True(t, processedOrder.Data.UpdatedAt.After(processedOrder.Data.CreatedAt), "UpdatedAt should be after CreatedAt")
		
		// Step 4: Verify order appears in customer orders
		customerOrders, err := getCustomerOrders(customerID)
		require.NoError(t, err, "Should retrieve customer orders successfully")
		
		found := false
		for _, order := range customerOrders.Data.Orders {
			if order.ID == orderID {
				found = true
				assert.Equal(t, models.OrderStatusCompleted, order.Status)
				break
			}
		}
		assert.True(t, found, "Order should appear in customer orders")
		
		// Step 5: Verify order appears in completed orders via status API
		completedOrders, err := getOrdersByStatus(models.OrderStatusCompleted)
		require.NoError(t, err, "Should retrieve completed orders successfully")
		
		found = false
		for _, order := range completedOrders.Data.Orders {
			if order.ID == orderID {
				found = true
				break
			}
		}
		assert.True(t, found, "Order should appear in completed orders")
		
		// Step 6: Verify order statistics are updated
		stats, err := getOrderStats()
		require.NoError(t, err, "Should retrieve order statistics successfully")
		
		assert.Greater(t, stats.Data.Total, 0, "Total orders should be greater than 0")
		assert.Greater(t, stats.Data.Completed, 0, "Completed orders should be greater than 0")
		
		t.Logf("Final stats: Total=%d, Completed=%d, Pending=%d", 
			stats.Data.Total, stats.Data.Completed, stats.Data.Pending)
	})
}

func TestOrderValidation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	err := waitForService(producerAPIURL + "/health")
	require.NoError(t, err, "Producer API should be available")
	
	t.Run("Invalid Order Requests", func(t *testing.T) {
		tests := []struct {
			name    string
			request models.CreateOrderRequest
			wantErr string
		}{
			{
				name: "invalid customer ID",
				request: models.CreateOrderRequest{
					CustomerID: uuid.Nil,
					Items: []models.CreateOrderItemRequest{
						{
							ProductID: uuid.New(),
							Name:      "Test Product",
							Price:     29.99,
							Quantity:  1,
						},
					},
				},
				wantErr: "invalid customer ID",
			},
			{
				name: "no items",
				request: models.CreateOrderRequest{
					CustomerID: uuid.New(),
					Items:      []models.CreateOrderItemRequest{},
				},
				wantErr: "at least one item is required",
			},
			{
				name: "invalid price",
				request: models.CreateOrderRequest{
					CustomerID: uuid.New(),
					Items: []models.CreateOrderItemRequest{
						{
							ProductID: uuid.New(),
							Name:      "Test Product",
							Price:     -1.0,
							Quantity:  1,
						},
					},
				},
				wantErr: "price must be greater than 0",
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := createOrder(tt.request)
				require.Error(t, err, "Should return validation error")
				assert.Contains(t, err.Error(), "400", "Should return 400 Bad Request")
			})
		}
	})
}

func TestMetrics_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}
	
	err := waitForService(statusAPIURL + "/health")
	require.NoError(t, err, "Status API should be available")
	
	t.Run("Metrics and Health Checks", func(t *testing.T) {
		// Test health endpoint
		resp, err := http.Get(statusAPIURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		var healthResp models.HealthResponse
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", healthResp.Status)
		assert.Equal(t, "order-processing-microservice", healthResp.Service)
		
		// Test metrics endpoint
		metricsResp, err := getMetrics()
		require.NoError(t, err)
		
		assert.NotNil(t, metricsResp.Data.Orders)
		assert.NotNil(t, metricsResp.Data.System)
		assert.NotEmpty(t, metricsResp.Data.System.Timestamp)
		
		// Verify order stats structure
		orderStats := metricsResp.Data.Orders
		assert.GreaterOrEqual(t, orderStats.Total, 0)
		assert.GreaterOrEqual(t, orderStats.Pending, 0)
		assert.GreaterOrEqual(t, orderStats.Processing, 0)
		assert.GreaterOrEqual(t, orderStats.Completed, 0)
		assert.GreaterOrEqual(t, orderStats.Failed, 0)
		assert.GreaterOrEqual(t, orderStats.Canceled, 0)
	})
}

// Helper functions

func waitForService(url string) error {
	timeout := 30 * time.Second
	interval := 2 * time.Second
	
	for elapsed := time.Duration(0); elapsed < timeout; elapsed += interval {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(interval)
	}
	
	return fmt.Errorf("service at %s not ready after %v", url, timeout)
}

func createOrder(req models.CreateOrderRequest) (*models.CreateOrderResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	
	resp, err := http.Post(producerAPIURL+"/api/v1/orders", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var orderResp models.CreateOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&orderResp)
	return &orderResp, err
}

func getOrder(orderID uuid.UUID) (*models.GetOrderResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/orders/%s", producerAPIURL, orderID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var orderResp models.GetOrderResponse
	err = json.NewDecoder(resp.Body).Decode(&orderResp)
	return &orderResp, err
}

func getCustomerOrders(customerID uuid.UUID) (*models.GetCustomerOrdersResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/orders/customer/%s", producerAPIURL, customerID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var ordersResp models.GetCustomerOrdersResponse
	err = json.NewDecoder(resp.Body).Decode(&ordersResp)
	return &ordersResp, err
}

func getOrdersByStatus(status models.OrderStatus) (*models.GetOrdersByStatusResponse, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/status/orders/%s", statusAPIURL, status))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var ordersResp models.GetOrdersByStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&ordersResp)
	return &ordersResp, err
}

func getOrderStats() (*models.GetOrderStatsResponse, error) {
	resp, err := http.Get(statusAPIURL + "/api/v1/status/stats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var statsResp models.GetOrderStatsResponse
	err = json.NewDecoder(resp.Body).Decode(&statsResp)
	return &statsResp, err
}

func getMetrics() (*models.GetMetricsResponse, error) {
	resp, err := http.Get(statusAPIURL + "/api/v1/status/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var metricsResp models.GetMetricsResponse
	err = json.NewDecoder(resp.Body).Decode(&metricsResp)
	return &metricsResp, err
}