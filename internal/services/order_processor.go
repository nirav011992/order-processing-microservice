package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/internal/queue"
	"order-processing-microservice/internal/repository"
)

type OrderProcessor struct {
	orderRepo repository.OrderRepository
	producer  queue.Producer
	logger    *logrus.Entry
}

func NewOrderProcessor(orderRepo repository.OrderRepository, producer queue.Producer) *OrderProcessor {
	return &OrderProcessor{
		orderRepo: orderRepo,
		producer:  producer,
		logger:    logrus.WithField("component", "order_processor"),
	}
}

func (p *OrderProcessor) HandleEvent(ctx context.Context, event *models.Event) error {
	switch event.Type {
	case models.OrderCreatedEvent:
		return p.handleOrderCreated(ctx, event)
	case models.OrderProcessingEvent:
		return p.handleOrderProcessing(ctx, event)
	default:
		p.logger.WithField("event_type", event.Type).Warn("Unhandled event type")
		return nil
	}
}

func (p *OrderProcessor) handleOrderCreated(ctx context.Context, event *models.Event) error {
	p.logger.WithField("event_id", event.ID).Info("Processing order created event")

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data format")
	}

	orderIDStr, ok := data["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid order_id in event data")
	}

	order, err := p.orderRepo.GetByID(ctx, parseUUID(orderIDStr))
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != models.OrderStatusPending {
		p.logger.WithFields(logrus.Fields{
			"order_id": order.ID,
			"status":   order.Status,
		}).Warn("Order is not in pending status, skipping processing")
		return nil
	}

	if err := p.orderRepo.UpdateStatus(ctx, order.ID, models.OrderStatusProcessing, order.Version); err != nil {
		return fmt.Errorf("failed to update order status to processing: %w", err)
	}

	processingEvent := models.NewOrderProcessingEvent(order)
	if err := p.producer.PublishEvent(ctx, processingEvent); err != nil {
		p.logger.WithError(err).Error("Failed to publish order processing event")
	}

	p.logger.WithField("order_id", order.ID).Info("Order moved to processing status")
	return nil
}

func (p *OrderProcessor) handleOrderProcessing(ctx context.Context, event *models.Event) error {
	p.logger.WithField("event_id", event.ID).Info("Processing order processing event")

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data format")
	}

	orderIDStr, ok := data["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid order_id in event data")
	}

	order, err := p.orderRepo.GetByID(ctx, parseUUID(orderIDStr))
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != models.OrderStatusProcessing {
		p.logger.WithFields(logrus.Fields{
			"order_id": order.ID,
			"status":   order.Status,
		}).Warn("Order is not in processing status, skipping")
		return nil
	}

	time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)

	success := rand.Float32() < 0.9

	if success {
		if err := p.orderRepo.UpdateStatus(ctx, order.ID, models.OrderStatusCompleted, order.Version); err != nil {
			return fmt.Errorf("failed to update order status to completed: %w", err)
		}

		completedEvent := models.NewOrderCompletedEvent(order)
		if err := p.producer.PublishEvent(ctx, completedEvent); err != nil {
			p.logger.WithError(err).Error("Failed to publish order completed event")
		}

		p.logger.WithField("order_id", order.ID).Info("Order completed successfully")
	} else {
		if err := p.orderRepo.UpdateStatus(ctx, order.ID, models.OrderStatusFailed, order.Version); err != nil {
			return fmt.Errorf("failed to update order status to failed: %w", err)
		}

		failedEvent := models.NewOrderFailedEvent(order, "Processing failed", "Random processing failure for simulation")
		if err := p.producer.PublishEvent(ctx, failedEvent); err != nil {
			p.logger.WithError(err).Error("Failed to publish order failed event")
		}

		p.logger.WithField("order_id", order.ID).Warn("Order processing failed")
	}

	return nil
}

func (p *OrderProcessor) ProcessPendingOrders(ctx context.Context) error {
	p.logger.Info("Processing pending orders")

	orders, err := p.orderRepo.GetByStatus(ctx, models.OrderStatusPending, 100, 0)
	if err != nil {
		return fmt.Errorf("failed to get pending orders: %w", err)
	}

	for _, order := range orders {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event := models.NewOrderCreatedEvent(order)
			if err := p.producer.PublishEvent(ctx, event); err != nil {
				p.logger.WithFields(logrus.Fields{
					"order_id": order.ID,
					"error":    err,
				}).Error("Failed to publish order created event for pending order")
				continue
			}
			
			p.logger.WithField("order_id", order.ID).Info("Republished event for pending order")
		}
	}

	p.logger.WithField("orders_processed", len(orders)).Info("Finished processing pending orders")
	return nil
}

func parseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}