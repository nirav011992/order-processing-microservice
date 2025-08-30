package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"order-processing-microservice/internal/models"
)

type PostgresOrderRepository struct {
	db     *sql.DB
	logger *logrus.Entry
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		db:     db,
		logger: logrus.WithField("component", "order_repository"),
	}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *models.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	order.CreatedAt = time.Now().UTC()
	order.UpdatedAt = order.CreatedAt
	order.Version = 1

	orderQuery := `
		INSERT INTO orders (id, customer_id, status, total_amount, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.ExecContext(ctx, orderQuery,
		order.ID, order.CustomerID, order.Status, order.TotalAmount,
		order.CreatedAt, order.UpdatedAt, order.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, quantity, price, total)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, item := range order.Items {
		item.ID = uuid.New()
		item.OrderID = order.ID
		item.Total = item.Price * float64(item.Quantity)

		_, err = tx.ExecContext(ctx, itemQuery,
			item.ID, item.OrderID, item.ProductID, item.Quantity, item.Price, item.Total,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.WithField("order_id", order.ID).Info("Order created successfully")
	return nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	orderQuery := `
		SELECT id, customer_id, status, total_amount, created_at, updated_at, version
		FROM orders
		WHERE id = $1
	`

	var order models.Order
	err := r.db.QueryRowContext(ctx, orderQuery, id).Scan(
		&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount,
		&order.CreatedAt, &order.UpdatedAt, &order.Version,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	itemsQuery := `
		SELECT id, order_id, product_id, quantity, price, total
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.Total)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	order.Items = items
	return &order, nil
}

func (r *PostgresOrderRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*models.Order, error) {
	query := `
		SELECT id, customer_id, status, total_amount, created_at, updated_at, version
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, customerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by customer ID: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount,
			&order.CreatedAt, &order.UpdatedAt, &order.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		items, err := r.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order items: %w", err)
		}
		order.Items = items
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *PostgresOrderRepository) Update(ctx context.Context, order *models.Order) error {
	order.UpdatedAt = time.Now().UTC()
	order.Version++

	query := `
		UPDATE orders
		SET status = $2, total_amount = $3, updated_at = $4, version = $5
		WHERE id = $1 AND version = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		order.ID, order.Status, order.TotalAmount, order.UpdatedAt, order.Version, order.Version-1,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found or version conflict")
	}

	r.logger.WithField("order_id", order.ID).Info("Order updated successfully")
	return nil
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus, version int) error {
	query := `
		UPDATE orders
		SET status = $2, updated_at = $3, version = $4
		WHERE id = $1 AND version = $5
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now().UTC(), version+1, version)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found or version conflict")
	}

	r.logger.WithFields(logrus.Fields{
		"order_id": id,
		"status":   status,
	}).Info("Order status updated successfully")
	return nil
}

func (r *PostgresOrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}

	r.logger.WithField("order_id", id).Info("Order deleted successfully")
	return nil
}

func (r *PostgresOrderRepository) GetByStatus(ctx context.Context, status models.OrderStatus, limit, offset int) ([]*models.Order, error) {
	query := `
		SELECT id, customer_id, status, total_amount, created_at, updated_at, version
		FROM orders
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by status: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount,
			&order.CreatedAt, &order.UpdatedAt, &order.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		items, err := r.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get order items: %w", err)
		}
		order.Items = items
		orders = append(orders, &order)
	}

	return orders, nil
}

func (r *PostgresOrderRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM orders`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return count, nil
}

func (r *PostgresOrderRepository) CountByStatus(ctx context.Context, status models.OrderStatus) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM orders WHERE status = $1`

	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders by status: %w", err)
	}

	return count, nil
}

func (r *PostgresOrderRepository) getOrderItems(ctx context.Context, orderID uuid.UUID) ([]models.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, quantity, price, total
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.Total)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}