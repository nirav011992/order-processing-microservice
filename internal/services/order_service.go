package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/internal/queue"
	"order-processing-microservice/internal/repository"
)

type OrderService struct {
	orderRepo repository.OrderRepository
	producer  queue.Producer
	logger    *logrus.Entry
}

func NewOrderService(orderRepo repository.OrderRepository, producer queue.Producer) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		producer:  producer,
		logger:    logrus.WithField("component", "order_service"),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	order := &models.Order{
		ID:         uuid.New(),
		CustomerID: req.CustomerID,
		Status:     models.OrderStatusPending,
		Items:      make([]models.OrderItem, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		orderItem := models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
		order.Items = append(order.Items, orderItem)
	}

	order.CalculateTotalAmount()

	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.WithError(err).Error("Failed to create order")
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	event := models.NewOrderCreatedEvent(order)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		s.logger.WithError(err).Error("Failed to publish order created event")
	}

	s.logger.WithField("order_id", order.ID).Info("Order created successfully")
	return order, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"order_id": id,
			"error":    err,
		}).Error("Failed to get order")
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrdersByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.Order, error) {
	orders, err := s.orderRepo.GetByCustomerID(ctx, customerID, limit, offset)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"customer_id": customerID,
			"error":       err,
		}).Error("Failed to get orders by customer ID")
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, newStatus models.OrderStatus, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if !order.IsValidStatusTransition(newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", order.Status, newStatus)
	}

	oldStatus := order.Status
	if err := s.orderRepo.UpdateStatus(ctx, id, newStatus, order.Version); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	order.Status = newStatus
	order.UpdatedAt = time.Now().UTC()
	order.Version++

	event := models.NewOrderStatusChangedEvent(order, oldStatus, reason)
	if err := s.producer.PublishEvent(ctx, event); err != nil {
		s.logger.WithError(err).Error("Failed to publish order status changed event")
	}

	s.logger.WithFields(logrus.Fields{
		"order_id":   id,
		"old_status": oldStatus,
		"new_status": newStatus,
	}).Info("Order status updated successfully")

	return nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id uuid.UUID, reason string) error {
	return s.UpdateOrderStatus(ctx, id, models.OrderStatusCanceled, reason)
}

func (s *OrderService) GetOrdersByStatus(ctx context.Context, status models.OrderStatus, limit, offset int) ([]*models.Order, error) {
	orders, err := s.orderRepo.GetByStatus(ctx, status, limit, offset)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"status": status,
			"error":  err,
		}).Error("Failed to get orders by status")
		return nil, fmt.Errorf("failed to get orders by status: %w", err)
	}

	return orders, nil
}

func (s *OrderService) GetOrderStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	totalCount, err := s.orderRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total order count: %w", err)
	}
	stats["total"] = totalCount

	statuses := []models.OrderStatus{
		models.OrderStatusPending,
		models.OrderStatusProcessing,
		models.OrderStatusCompleted,
		models.OrderStatusCanceled,
		models.OrderStatusFailed,
	}

	for _, status := range statuses {
		count, err := s.orderRepo.CountByStatus(ctx, status)
		if err != nil {
			return nil, fmt.Errorf("failed to get count for status %s: %w", status, err)
		}
		stats[string(status)] = count
	}

	return stats, nil
}