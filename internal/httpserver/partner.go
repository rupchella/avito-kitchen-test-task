package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const (
	maxExternalIDLength  = 255
	maxItemNameLength    = 255
	maxDescriptionLength = 2000
	maxMenuItems         = 1000
)

type PartnerService interface {
	Authenticate(
		ctx context.Context,
		apiKey string,
	) (string, error)

	SyncMenu(
		ctx context.Context,
		restaurantID string,
		inputs []domain.MenuItemInput,
	) ([]domain.MenuItem, error)

	SetMenuItemAvailability(
		ctx context.Context,
		restaurantID string,
		externalID string,
		isAvailable bool,
	) (domain.MenuItem, error)

	ListOrders(
		ctx context.Context,
		restaurantID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)

	UpdateOrderStatus(
		ctx context.Context,
		restaurantID string,
		orderID string,
		target domain.OrderStatus,
	) (domain.Order, error)
}

type PartnerHandler struct {
	service PartnerService
	logger  *slog.Logger
}

func NewPartnerHandler(
	service PartnerService,
	logger *slog.Logger,
) *PartnerHandler {
	return &PartnerHandler{
		service: service,
		logger:  logger,
	}
}

type partnerHandlerFunc func(
	writer http.ResponseWriter,
	request *http.Request,
	restaurantID string,
)

func (h *PartnerHandler) Authenticated(
	next partnerHandlerFunc,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		restaurantID, err := h.service.Authenticate(
			request.Context(),
			request.Header.Get(apiKeyHeader),
		)
		if err != nil {
			writeError(writer, h.logger, "failed to authenticate partner", err)

			return
		}

		next(writer, request, restaurantID)
	}
}

func (h *PartnerHandler) SyncMenu(
	writer http.ResponseWriter,
	request *http.Request,
	restaurantID string,
) {
	var body syncMenuRequest

	if err := decodeJSON(writer, request, &body); err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	inputs, err := body.toDomain()
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	items, err := h.service.SyncMenu(request.Context(), restaurantID, inputs)
	if err != nil {
		writeError(writer, h.logger, "failed to sync menu", err)

		return
	}

	writeJSON(writer, http.StatusOK, menuSyncResponse{
		RestaurantID: restaurantID,
		Items:        newMenuItemResponses(items),
		UpdatedAt:    time.Now().UTC(),
	})
}

func (h *PartnerHandler) SetMenuItemAvailability(
	writer http.ResponseWriter,
	request *http.Request,
	restaurantID string,
) {
	externalID := request.PathValue("externalId")

	if externalID == "" || len([]rune(externalID)) > maxExternalIDLength {
		writeValidationError(writer, fmt.Sprintf(
			"externalId must contain from 1 to %d characters",
			maxExternalIDLength,
		))

		return
	}

	var body updateAvailabilityRequest

	if err := decodeJSON(writer, request, &body); err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	if body.IsAvailable == nil {
		writeValidationError(writer, "is_available is required")

		return
	}

	item, err := h.service.SetMenuItemAvailability(
		request.Context(),
		restaurantID,
		externalID,
		*body.IsAvailable,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to update menu item", err)

		return
	}

	writeJSON(writer, http.StatusOK, newMenuItemResponse(item))
}

func (h *PartnerHandler) ListOrders(
	writer http.ResponseWriter,
	request *http.Request,
	restaurantID string,
) {
	status, err := parseOrderStatusFilter(request, domain.OrderStatus.IsValid)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	if status == nil {
		defaultStatus := domain.OrderStatusCreated
		status = &defaultStatus
	}

	page, err := parsePagination(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	orders, total, err := h.service.ListOrders(
		request.Context(),
		restaurantID,
		status,
		page.limit,
		page.offset,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to list partner orders", err)

		return
	}

	writeJSON(writer, http.StatusOK, orderListResponse{
		Items:  newOrderResponses(orders),
		Total:  total,
		Limit:  page.limit,
		Offset: page.offset,
	})
}

func (h *PartnerHandler) UpdateOrderStatus(
	writer http.ResponseWriter,
	request *http.Request,
	restaurantID string,
) {
	orderID, err := parseUUIDPathValue(request, "orderId")
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	var body updateOrderStatusRequest

	if err := decodeJSON(writer, request, &body); err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	status := domain.OrderStatus(body.Status)

	if !isPartnerOrderStatus(status) {
		writeValidationError(
			writer,
			"status must be one of: accepted, rejected, preparing, ready",
		)

		return
	}

	order, err := h.service.UpdateOrderStatus(
		request.Context(),
		restaurantID,
		orderID,
		status,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to update order status", err)

		return
	}

	writeJSON(writer, http.StatusOK, newOrderResponse(order))
}

type partnerMenuItemRequest struct {
	ExternalID   string `json:"external_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceKopecks int64  `json:"price_kopecks"`
	IsAvailable  *bool  `json:"is_available"`
}

type syncMenuRequest struct {
	Items []partnerMenuItemRequest `json:"items"`
}

func (r syncMenuRequest) toDomain() ([]domain.MenuItemInput, error) {
	if len(r.Items) > maxMenuItems {
		return nil, fmt.Errorf(
			"items must contain at most %d positions",
			maxMenuItems,
		)
	}

	inputs := make([]domain.MenuItemInput, 0, len(r.Items))

	for _, item := range r.Items {
		if err := item.validate(); err != nil {
			return nil, err
		}

		inputs = append(inputs, domain.MenuItemInput{
			ExternalID:   item.ExternalID,
			Name:         item.Name,
			Description:  item.Description,
			PriceKopecks: item.PriceKopecks,
			IsAvailable:  *item.IsAvailable,
		})
	}

	return inputs, nil
}

func (r partnerMenuItemRequest) validate() error {
	if err := checkLength(
		"external_id",
		r.ExternalID,
		1,
		maxExternalIDLength,
	); err != nil {
		return err
	}

	if err := checkLength("name", r.Name, 1, maxItemNameLength); err != nil {
		return err
	}

	if err := checkLength(
		"description",
		r.Description,
		0,
		maxDescriptionLength,
	); err != nil {
		return err
	}

	if r.PriceKopecks < 1 {
		return fmt.Errorf("price_kopecks must be a positive integer")
	}

	if r.IsAvailable == nil {
		return fmt.Errorf("is_available is required")
	}

	return nil
}

type updateAvailabilityRequest struct {
	IsAvailable *bool `json:"is_available"`
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

type menuSyncResponse struct {
	RestaurantID string             `json:"restaurant_id"`
	Items        []menuItemResponse `json:"items"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

func isPartnerOrderStatus(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusAccepted,
		domain.OrderStatusRejected,
		domain.OrderStatusPreparing,
		domain.OrderStatusReady:
		return true
	default:
		return false
	}
}

func checkLength(field, value string, minimum, maximum int) error {
	length := len([]rune(value))

	if length < minimum || length > maximum {
		return fmt.Errorf(
			"%s must contain from %d to %d characters",
			field,
			minimum,
			maximum,
		)
	}

	return nil
}
