package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/service"
)

const (
	minDeliveryAddressLength = 5
	maxDeliveryAddressLength = 1000
	maxOrderItems            = 100
	maxItemQuantity          = 100
)

type OrderService interface {
	Create(
		ctx context.Context,
		input service.CreateOrderInput,
	) (domain.Order, bool, error)

	Get(
		ctx context.Context,
		orderID string,
		userID string,
	) (domain.Order, error)

	List(
		ctx context.Context,
		userID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)
}

type OrderHandler struct {
	service OrderService
	logger  *slog.Logger
}

func NewOrderHandler(
	service OrderService,
	logger *slog.Logger,
) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger,
	}
}

func (h *OrderHandler) Create(
	writer http.ResponseWriter,
	request *http.Request,
) {
	userID, err := parseUserID(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	idempotencyKey, err := parseIdempotencyKey(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	var body createOrderRequest

	if err := decodeJSON(writer, request, &body); err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	if err := body.validate(); err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	input := service.CreateOrderInput{
		UserID:          userID,
		IdempotencyKey:  idempotencyKey,
		RestaurantID:    body.RestaurantID,
		DeliveryAddress: body.DeliveryAddress,
		Items:           make([]service.CreateOrderItem, 0, len(body.Items)),
	}

	for _, item := range body.Items {
		input.Items = append(input.Items, service.CreateOrderItem{
			MenuItemID: item.MenuItemID,
			Quantity:   item.Quantity,
		})
	}

	order, created, err := h.service.Create(request.Context(), input)
	if err != nil {
		writeError(writer, h.logger, "failed to create order", err)

		return
	}

	if !created {
		writeJSON(writer, http.StatusOK, newOrderResponse(order))

		return
	}

	writer.Header().Set("Location", "/api/v1/orders/"+order.ID)
	writeJSON(writer, http.StatusCreated, newOrderResponse(order))
}

func (h *OrderHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
	userID, err := parseUserID(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	orderID, err := parseUUIDPathValue(request, "orderId")
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	order, err := h.service.Get(request.Context(), orderID, userID)
	if err != nil {
		writeError(writer, h.logger, "failed to get order", err)

		return
	}

	writeJSON(writer, http.StatusOK, newOrderResponse(order))
}

func (h *OrderHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	userID, err := parseUserID(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	status, err := parseOrderStatusFilter(request, domain.OrderStatus.IsValid)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	page, err := parsePagination(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	orders, total, err := h.service.List(
		request.Context(),
		userID,
		status,
		page.limit,
		page.offset,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to list orders", err)

		return
	}

	writeJSON(writer, http.StatusOK, orderListResponse{
		Items:  newOrderResponses(orders),
		Total:  total,
		Limit:  page.limit,
		Offset: page.offset,
	})
}

type createOrderItemRequest struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type createOrderRequest struct {
	RestaurantID    string                   `json:"restaurant_id"`
	DeliveryAddress string                   `json:"delivery_address"`
	Items           []createOrderItemRequest `json:"items"`
}

func (r createOrderRequest) validate() error {
	if err := uuid.Validate(r.RestaurantID); err != nil {
		return fmt.Errorf("restaurant_id must be a valid UUID")
	}

	addressLength := len([]rune(r.DeliveryAddress))

	if addressLength < minDeliveryAddressLength ||
		addressLength > maxDeliveryAddressLength {
		return fmt.Errorf(
			"delivery_address must contain from %d to %d characters",
			minDeliveryAddressLength,
			maxDeliveryAddressLength,
		)
	}

	if len(r.Items) == 0 || len(r.Items) > maxOrderItems {
		return fmt.Errorf(
			"items must contain from 1 to %d positions",
			maxOrderItems,
		)
	}

	for _, item := range r.Items {
		if err := uuid.Validate(item.MenuItemID); err != nil {
			return fmt.Errorf("menu_item_id must be a valid UUID")
		}

		if item.Quantity < 1 || item.Quantity > maxItemQuantity {
			return fmt.Errorf(
				"quantity must be an integer from 1 to %d",
				maxItemQuantity,
			)
		}
	}

	return nil
}

type orderItemResponse struct {
	MenuItemID        string `json:"menu_item_id"`
	ItemName          string `json:"item_name"`
	UnitPriceKopecks  int64  `json:"unit_price_kopecks"`
	Quantity          int    `json:"quantity"`
	TotalPriceKopecks int64  `json:"total_price_kopecks"`
}

type orderResponse struct {
	ID                string              `json:"id"`
	RestaurantID      string              `json:"restaurant_id"`
	UserID            string              `json:"user_id"`
	Status            domain.OrderStatus  `json:"status"`
	TotalPriceKopecks int64               `json:"total_price_kopecks"`
	DeliveryAddress   string              `json:"delivery_address"`
	Items             []orderItemResponse `json:"items"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type orderListResponse struct {
	Items  []orderResponse `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func newOrderResponse(order domain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, orderItemResponse{
			MenuItemID:        item.MenuItemID,
			ItemName:          item.ItemName,
			UnitPriceKopecks:  item.UnitPriceKopecks,
			Quantity:          item.Quantity,
			TotalPriceKopecks: item.TotalPriceKopecks(),
		})
	}

	return orderResponse{
		ID:                order.ID,
		RestaurantID:      order.RestaurantID,
		UserID:            order.UserID,
		Status:            order.Status,
		TotalPriceKopecks: order.TotalPriceKopecks,
		DeliveryAddress:   order.DeliveryAddress,
		Items:             items,
		CreatedAt:         order.CreatedAt,
		UpdatedAt:         order.UpdatedAt,
	}
}

func newOrderResponses(orders []domain.Order) []orderResponse {
	responses := make([]orderResponse, 0, len(orders))

	for _, order := range orders {
		responses = append(responses, newOrderResponse(order))
	}

	return responses
}
