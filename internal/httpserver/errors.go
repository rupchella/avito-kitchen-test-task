package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const maxRequestBodyBytes = 1 << 20

var domainErrorMapping = []struct {
	err        error
	statusCode int
	code       string
}{
	{domain.ErrRestaurantNotFound, http.StatusNotFound, "restaurant_not_found"},
	{domain.ErrMenuItemNotFound, http.StatusNotFound, "menu_item_not_found"},
	{domain.ErrOrderNotFound, http.StatusNotFound, "order_not_found"},
	{domain.ErrMenuItemUnavailable, http.StatusConflict, "menu_item_unavailable"},
	{domain.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
	{domain.ErrOrderStatusForbidden, http.StatusConflict, "order_status_forbidden"},
	{domain.ErrDuplicateMenuItem, http.StatusBadRequest, "validation_error"},
	{domain.ErrPartnerUnauthorized, http.StatusUnauthorized, "unauthorized"},
}

func writeValidationError(
	writer http.ResponseWriter,
	message string,
) {
	writeJSON(writer, http.StatusBadRequest, errorResponse{
		Code:    "validation_error",
		Message: message,
	})
}

func writeError(
	writer http.ResponseWriter,
	logger *slog.Logger,
	operation string,
	err error,
) {
	for _, mapping := range domainErrorMapping {
		if errors.Is(err, mapping.err) {
			writeJSON(writer, mapping.statusCode, errorResponse{
				Code:    mapping.code,
				Message: mapping.err.Error(),
			})

			return
		}
	}

	logger.Error(operation, "error", err)

	writeJSON(writer, http.StatusInternalServerError, errorResponse{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

func decodeJSON(
	writer http.ResponseWriter,
	request *http.Request,
	target any,
) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("request body is not a valid JSON document: %w", err)
	}

	return nil
}
