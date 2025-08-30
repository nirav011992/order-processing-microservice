package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCanceled   OrderStatus = "canceled"
	OrderStatusFailed     OrderStatus = "failed"
)

type Order struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	CustomerID  uuid.UUID   `json:"customer_id" db:"customer_id" binding:"required"`
	Status      OrderStatus `json:"status" db:"status"`
	Items       []OrderItem `json:"items" binding:"required,min=1"`
	TotalAmount float64     `json:"total_amount" db:"total_amount"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
	Version     int         `json:"version" db:"version"`
}

type OrderItem struct {
	ID        uuid.UUID `json:"id" db:"id"`
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" db:"quantity" binding:"required,min=1"`
	Price     float64   `json:"price" db:"price" binding:"required,min=0"`
	Total     float64   `json:"total" db:"total"`
}

type CreateOrderRequest struct {
	CustomerID uuid.UUID               `json:"customer_id" binding:"required"`
	Items      []CreateOrderItemRequest `json:"items" binding:"required,min=1"`
}

type CreateOrderItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
	Price     float64   `json:"price" binding:"required,min=0"`
}

type OrderResponse struct {
	ID          uuid.UUID   `json:"id"`
	CustomerID  uuid.UUID   `json:"customer_id"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (o *Order) CalculateTotalAmount() {
	total := 0.0
	for _, item := range o.Items {
		item.Total = item.Price * float64(item.Quantity)
		total += item.Total
	}
	o.TotalAmount = total
}

func (o *Order) IsValidStatusTransition(newStatus OrderStatus) bool {
	validTransitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:    {OrderStatusProcessing, OrderStatusCanceled},
		OrderStatusProcessing: {OrderStatusCompleted, OrderStatusFailed, OrderStatusCanceled},
		OrderStatusCompleted:  {},
		OrderStatusCanceled:   {},
		OrderStatusFailed:     {OrderStatusPending},
	}

	allowedStatuses, exists := validTransitions[o.Status]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowedStatuses {
		if allowedStatus == newStatus {
			return true
		}
	}
	return false
}