package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"order-processing-microservice/internal/models"
	"order-processing-microservice/internal/services"
	"order-processing-microservice/pkg/utils"
)

type ProducerHandlers struct {
	orderService *services.OrderService
}

func NewProducerHandlers(orderService *services.OrderService) *ProducerHandlers {
	return &ProducerHandlers{
		orderService: orderService,
	}
}

func (h *ProducerHandlers) CreateOrder(c *gin.Context) {
	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithValidationError(c, err)
		return
	}

	if len(req.Items) == 0 {
		utils.RespondWithError(c, http.StatusBadRequest, 
			fmt.Errorf("at least one item is required"), "Order must contain at least one item")
		return
	}

	order, err := h.orderService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		utils.RespondWithInternalError(c, err)
		return
	}

	response := &models.OrderResponse{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		Status:      order.Status,
		Items:       order.Items,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}

	utils.RespondWithCreated(c, response, "Order created successfully")
}

func (h *ProducerHandlers) GetOrder(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err, "Invalid order ID format")
		return
	}

	order, err := h.orderService.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "order not found" {
			utils.RespondWithNotFound(c, "Order")
			return
		}
		utils.RespondWithInternalError(c, err)
		return
	}

	response := &models.OrderResponse{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		Status:      order.Status,
		Items:       order.Items,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}

	utils.RespondWithSuccess(c, response)
}

func (h *ProducerHandlers) GetOrdersByCustomer(c *gin.Context) {
	customerIDParam := c.Param("customerId")
	customerID, err := uuid.Parse(customerIDParam)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err, "Invalid customer ID format")
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

	orders, err := h.orderService.GetOrdersByCustomerID(c.Request.Context(), customerID, limit, offset)
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

	utils.RespondWithSuccess(c, responses)
}

func (h *ProducerHandlers) UpdateOrderStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err, "Invalid order ID format")
		return
	}

	var req struct {
		Status models.OrderStatus `json:"status" binding:"required"`
		Reason string             `json:"reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithValidationError(c, err)
		return
	}

	if err := h.orderService.UpdateOrderStatus(c.Request.Context(), id, req.Status, req.Reason); err != nil {
		if err.Error() == "order not found" {
			utils.RespondWithNotFound(c, "Order")
			return
		}
		utils.RespondWithError(c, http.StatusBadRequest, err)
		return
	}

	utils.RespondWithSuccess(c, nil, "Order status updated successfully")
}

func (h *ProducerHandlers) CancelOrder(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err, "Invalid order ID format")
		return
	}

	var req struct {
		Reason string `json:"reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Cancelled by user"
	}

	if err := h.orderService.CancelOrder(c.Request.Context(), id, req.Reason); err != nil {
		if err.Error() == "order not found" {
			utils.RespondWithNotFound(c, "Order")
			return
		}
		utils.RespondWithError(c, http.StatusBadRequest, err)
		return
	}

	utils.RespondWithSuccess(c, nil, "Order cancelled successfully")
}

func (h *ProducerHandlers) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		orders := api.Group("/orders")
		{
			orders.POST("", h.CreateOrder)
			orders.GET("/:id", h.GetOrder)
			orders.PUT("/:id/status", h.UpdateOrderStatus)
			orders.PUT("/:id/cancel", h.CancelOrder)
		}

		customers := api.Group("/customers")
		{
			customers.GET("/:customerId/orders", h.GetOrdersByCustomer)
		}
	}
}