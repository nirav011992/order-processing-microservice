package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	OrderCreatedEvent        EventType = "order.created"
	OrderStatusChangedEvent  EventType = "order.status.changed"
	OrderProcessingEvent     EventType = "order.processing"
	OrderCompletedEvent      EventType = "order.completed"
	OrderFailedEvent         EventType = "order.failed"
	OrderCanceledEvent       EventType = "order.canceled"
)

type Event struct {
	ID        uuid.UUID   `json:"id"`
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Version   string      `json:"version"`
}

type OrderCreatedEventData struct {
	OrderID     uuid.UUID   `json:"order_id"`
	CustomerID  uuid.UUID   `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
}

type OrderStatusChangedEventData struct {
	OrderID     uuid.UUID   `json:"order_id"`
	CustomerID  uuid.UUID   `json:"customer_id"`
	OldStatus   OrderStatus `json:"old_status"`
	NewStatus   OrderStatus `json:"new_status"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Reason      string      `json:"reason,omitempty"`
}

type OrderProcessingEventData struct {
	OrderID    uuid.UUID `json:"order_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	StartedAt  time.Time `json:"started_at"`
}

type OrderCompletedEventData struct {
	OrderID     uuid.UUID `json:"order_id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	CompletedAt time.Time `json:"completed_at"`
	TotalAmount float64   `json:"total_amount"`
}

type OrderFailedEventData struct {
	OrderID    uuid.UUID `json:"order_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	FailedAt   time.Time `json:"failed_at"`
	Reason     string    `json:"reason"`
	Error      string    `json:"error,omitempty"`
}

type OrderCanceledEventData struct {
	OrderID     uuid.UUID `json:"order_id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	CanceledAt  time.Time `json:"canceled_at"`
	Reason      string    `json:"reason,omitempty"`
}

func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		ID:        uuid.New(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}
}

func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Event) FromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}

func NewOrderCreatedEvent(order *Order) *Event {
	data := OrderCreatedEventData{
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		Items:       order.Items,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
	}
	return NewEvent(OrderCreatedEvent, data)
}

func NewOrderStatusChangedEvent(order *Order, oldStatus OrderStatus, reason string) *Event {
	data := OrderStatusChangedEventData{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		OldStatus:  oldStatus,
		NewStatus:  order.Status,
		UpdatedAt:  order.UpdatedAt,
		Reason:     reason,
	}
	return NewEvent(OrderStatusChangedEvent, data)
}

func NewOrderProcessingEvent(order *Order) *Event {
	data := OrderProcessingEventData{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		StartedAt:  time.Now().UTC(),
	}
	return NewEvent(OrderProcessingEvent, data)
}

func NewOrderCompletedEvent(order *Order) *Event {
	data := OrderCompletedEventData{
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		CompletedAt: time.Now().UTC(),
		TotalAmount: order.TotalAmount,
	}
	return NewEvent(OrderCompletedEvent, data)
}

func NewOrderFailedEvent(order *Order, reason, errorMsg string) *Event {
	data := OrderFailedEventData{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		FailedAt:   time.Now().UTC(),
		Reason:     reason,
		Error:      errorMsg,
	}
	return NewEvent(OrderFailedEvent, data)
}

func NewOrderCanceledEvent(order *Order, reason string) *Event {
	data := OrderCanceledEventData{
		OrderID:    order.ID,
		CustomerID: order.CustomerID,
		CanceledAt: time.Now().UTC(),
		Reason:     reason,
	}
	return NewEvent(OrderCanceledEvent, data)
}