package repository

import (
	"context"
	"order-processing-microservice/internal/models"
	"github.com/google/uuid"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.Order, error)
	Update(ctx context.Context, order *models.Order) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus, version int) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByStatus(ctx context.Context, status models.OrderStatus, limit, offset int) ([]*models.Order, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status models.OrderStatus) (int64, error)
}