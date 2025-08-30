package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/internal/services"
	"order-processing-microservice/pkg/config"
)

// Mock implementations
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.Order, error) {
	args := m.Called(ctx, customerID, limit, offset)
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus, version int) error {
	args := m.Called(ctx, id, status, version)
	return args.Error(0)
}

func (m *MockOrderRepository) GetOrderStats(ctx context.Context) (*models.OrderStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*models.OrderStats), args.Error(1)
}

func (m *MockOrderRepository) GetOrdersByStatus(ctx context.Context, status models.OrderStatus, limit, offset int) ([]*models.Order, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetPendingOrders(ctx context.Context, limit int) ([]*models.Order, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*models.Order), args.Error(1)
}

type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) PublishEvent(ctx context.Context, event *models.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestOrderService_CreateOrder(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}
	mockProducer := &MockProducer{}
	
	service := services.NewOrderService(mockRepo, mockProducer)
	
	tests := []struct {
		name      string
		request   *models.CreateOrderRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful order creation",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     29.99,
						Quantity:  2,
					},
				},
			},
			setupMock: func() {
				mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Order")).Return(nil)
				mockProducer.On("PublishEvent", ctx, mock.AnythingOfType("*models.Event")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "repository error",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     29.99,
						Quantity:  2,
					},
				},
			},
			setupMock: func() {
				mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Order")).Return(errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "invalid request - no items",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items:      []models.CreateOrderItemRequest{},
			},
			setupMock: func() {
				// No mocks needed as validation should fail
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepo.ExpectedCalls = nil
			mockProducer.ExpectedCalls = nil
			
			tt.setupMock()
			
			order, err := service.CreateOrder(ctx, tt.request)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.Equal(t, tt.request.CustomerID, order.CustomerID)
				assert.Equal(t, models.OrderStatusPending, order.Status)
				assert.Equal(t, len(tt.request.Items), len(order.Items))
				
				// Verify total amount calculation
				expectedTotal := tt.request.Items[0].Price * float64(tt.request.Items[0].Quantity)
				assert.Equal(t, expectedTotal, order.TotalAmount)
			}
			
			mockRepo.AssertExpectations(t)
			mockProducer.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetOrder(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}
	mockProducer := &MockProducer{}
	
	service := services.NewOrderService(mockRepo, mockProducer)
	
	orderID := uuid.New()
	expectedOrder := &models.Order{
		ID:         orderID,
		CustomerID: uuid.New(),
		Status:     models.OrderStatusPending,
		Items: []models.OrderItem{
			{
				ID:        uuid.New(),
				OrderID:   orderID,
				ProductID: uuid.New(),
				Quantity:  2,
				Price:     29.99,
				Total:     59.98,
			},
		},
		TotalAmount: 59.98,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}
	
	tests := []struct {
		name      string
		orderID   uuid.UUID
		setupMock func()
		expected  *models.Order
		wantErr   bool
	}{
		{
			name:    "successful order retrieval",
			orderID: orderID,
			setupMock: func() {
				mockRepo.On("GetByID", ctx, orderID).Return(expectedOrder, nil)
			},
			expected: expectedOrder,
			wantErr:  false,
		},
		{
			name:    "order not found",
			orderID: orderID,
			setupMock: func() {
				mockRepo.On("GetByID", ctx, orderID).Return((*models.Order)(nil), errors.New("order not found"))
			},
			expected: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepo.ExpectedCalls = nil
			
			tt.setupMock()
			
			order, err := service.GetOrder(ctx, tt.orderID)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, order)
			}
			
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetOrdersByCustomer(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}
	mockProducer := &MockProducer{}
	
	service := services.NewOrderService(mockRepo, mockProducer)
	
	customerID := uuid.New()
	expectedOrders := []*models.Order{
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			Status:      models.OrderStatusCompleted,
			TotalAmount: 59.98,
		},
		{
			ID:          uuid.New(),
			CustomerID:  customerID,
			Status:      models.OrderStatusPending,
			TotalAmount: 29.99,
		},
	}
	
	tests := []struct {
		name       string
		customerID uuid.UUID
		limit      int
		offset     int
		setupMock  func()
		expected   []*models.Order
		wantErr    bool
	}{
		{
			name:       "successful retrieval",
			customerID: customerID,
			limit:      10,
			offset:     0,
			setupMock: func() {
				mockRepo.On("GetByCustomerID", ctx, customerID, 10, 0).Return(expectedOrders, nil)
			},
			expected: expectedOrders,
			wantErr:  false,
		},
		{
			name:       "repository error",
			customerID: customerID,
			limit:      10,
			offset:     0,
			setupMock: func() {
				mockRepo.On("GetByCustomerID", ctx, customerID, 10, 0).Return([]*models.Order(nil), errors.New("database error"))
			},
			expected: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepo.ExpectedCalls = nil
			
			tt.setupMock()
			
			orders, err := service.GetOrdersByCustomer(ctx, tt.customerID, tt.limit, tt.offset)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, orders)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, orders)
			}
			
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}
	mockProducer := &MockProducer{}
	
	service := services.NewOrderService(mockRepo, mockProducer)
	
	orderID := uuid.New()
	
	tests := []struct {
		name      string
		orderID   uuid.UUID
		status    models.OrderStatus
		version   int
		setupMock func()
		wantErr   bool
	}{
		{
			name:    "successful status update",
			orderID: orderID,
			status:  models.OrderStatusProcessing,
			version: 1,
			setupMock: func() {
				mockRepo.On("UpdateStatus", ctx, orderID, models.OrderStatusProcessing, 1).Return(nil)
				mockProducer.On("PublishEvent", ctx, mock.AnythingOfType("*models.Event")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "repository error",
			orderID: orderID,
			status:  models.OrderStatusProcessing,
			version: 1,
			setupMock: func() {
				mockRepo.On("UpdateStatus", ctx, orderID, models.OrderStatusProcessing, 1).Return(errors.New("database error"))
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepo.ExpectedCalls = nil
			mockProducer.ExpectedCalls = nil
			
			tt.setupMock()
			
			err := service.UpdateOrderStatus(ctx, tt.orderID, tt.status, tt.version)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			mockRepo.AssertExpectations(t)
			mockProducer.AssertExpectations(t)
		})
	}
}

func TestOrderService_GetOrderStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockOrderRepository{}
	mockProducer := &MockProducer{}
	
	service := services.NewOrderService(mockRepo, mockProducer)
	
	expectedStats := &models.OrderStats{
		Pending:    5,
		Processing: 3,
		Completed:  10,
		Failed:     1,
		Canceled:   2,
		Total:      21,
	}
	
	tests := []struct {
		name      string
		setupMock func()
		expected  *models.OrderStats
		wantErr   bool
	}{
		{
			name: "successful stats retrieval",
			setupMock: func() {
				mockRepo.On("GetOrderStats", ctx).Return(expectedStats, nil)
			},
			expected: expectedStats,
			wantErr:  false,
		},
		{
			name: "repository error",
			setupMock: func() {
				mockRepo.On("GetOrderStats", ctx).Return((*models.OrderStats)(nil), errors.New("database error"))
			},
			expected: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mockRepo.ExpectedCalls = nil
			
			tt.setupMock()
			
			stats, err := service.GetOrderStats(ctx)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, stats)
			}
			
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestOrderService_ValidateOrderRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *models.CreateOrderRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     29.99,
						Quantity:  2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty customer ID",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.Nil,
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     29.99,
						Quantity:  2,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no items",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items:      []models.CreateOrderItemRequest{},
			},
			wantErr: true,
		},
		{
			name: "invalid item price",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     -1.0,
						Quantity:  2,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid item quantity",
			request: &models.CreateOrderRequest{
				CustomerID: uuid.New(),
				Items: []models.CreateOrderItemRequest{
					{
						ProductID: uuid.New(),
						Name:      "Test Product",
						Price:     29.99,
						Quantity:  0,
					},
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := services.ValidateCreateOrderRequest(tt.request)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}