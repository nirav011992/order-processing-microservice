package queue

import (
	"context"
	"order-processing-microservice/internal/models"
)

type Producer interface {
	PublishEvent(ctx context.Context, event *models.Event) error
	Close() error
}

type Consumer interface {
	Subscribe(ctx context.Context, handler EventHandler) error
	Close() error
}

type EventHandler interface {
	HandleEvent(ctx context.Context, event *models.Event) error
}

type EventHandlerFunc func(ctx context.Context, event *models.Event) error

func (f EventHandlerFunc) HandleEvent(ctx context.Context, event *models.Event) error {
	return f(ctx, event)
}