package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/internal/services"
	"order-processing-microservice/pkg/utils"
)

type StatusHandlers struct {
	orderService *services.OrderService
}

func NewStatusHandlers(orderService *services.OrderService) *StatusHandlers {
	return &StatusHandlers{
		orderService: orderService,
	}
}

func (h *StatusHandlers) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "order-processing-microservice",
		"version":   "1.0.0",
	}

	c.JSON(http.StatusOK, health)
}

func (h *StatusHandlers) GetOrderStats(c *gin.Context) {
	stats, err := h.orderService.GetOrderStats(c.Request.Context())
	if err != nil {
		utils.RespondWithInternalError(c, err)
		return
	}

	utils.RespondWithSuccess(c, stats)
}

func (h *StatusHandlers) GetOrdersByStatus(c *gin.Context) {
	statusParam := c.Param("status")
	status := models.OrderStatus(statusParam)

	validStatuses := map[models.OrderStatus]bool{
		models.OrderStatusPending:    true,
		models.OrderStatusProcessing: true,
		models.OrderStatusCompleted:  true,
		models.OrderStatusCanceled:   true,
		models.OrderStatusFailed:     true,
	}

	if !validStatuses[status] {
		utils.RespondWithError(c, http.StatusBadRequest, 
			fmt.Errorf("invalid status"), "Valid statuses: pending, processing, completed, canceled, failed")
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	orders, err := h.orderService.GetOrdersByStatus(c.Request.Context(), status, limit, offset)
	if err != nil {
		utils.RespondWithInternalError(c, err)
		return
	}

	var responses []*models.OrderResponse
	for _, order := range orders {
		response := &models.OrderResponse{
			ID:          order.ID,
			CustomerID:  order.CustomerID,
			Status:      order.Status,
			Items:       order.Items,
			TotalAmount: order.TotalAmount,
			CreatedAt:   order.CreatedAt,
			UpdatedAt:   order.UpdatedAt,
		}
		responses = append(responses, response)
	}

	responseData := gin.H{
		"orders": responses,
		"meta": gin.H{
			"status": status,
			"limit":  limit,
			"offset": offset,
			"count":  len(responses),
		},
	}

	utils.RespondWithSuccess(c, responseData)
}

func (h *StatusHandlers) GetMetrics(c *gin.Context) {
	stats, err := h.orderService.GetOrderStats(c.Request.Context())
	if err != nil {
		utils.RespondWithInternalError(c, err)
		return
	}

	metrics := gin.H{
		"orders": stats,
		"system": gin.H{
			"uptime":    time.Since(time.Now().Add(-time.Hour)).String(), // Placeholder
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	utils.RespondWithSuccess(c, metrics)
}

func (h *StatusHandlers) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.HealthCheck)
	
	api := r.Group("/api/v1")
	{
		status := api.Group("/status")
		{
			status.GET("/stats", h.GetOrderStats)
			status.GET("/orders/:status", h.GetOrdersByStatus)
			status.GET("/metrics", h.GetMetrics)
		}
	}
}