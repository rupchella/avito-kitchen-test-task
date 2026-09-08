package httpserver

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const (
	defaultLimit  = 20
	maxLimit      = 100
	defaultOffset = 0

	userIDHeader         = "X-User-ID"
	idempotencyKeyHeader = "Idempotency-Key"
	apiKeyHeader         = "X-API-Key" //nolint:gosec // имя заголовка, а не ключ

	maxIdempotencyKeyLength = 128
)

type pagination struct {
	limit  int
	offset int
}

func parsePagination(request *http.Request) (pagination, error) {
	result := pagination{
		limit:  defaultLimit,
		offset: defaultOffset,
	}

	if rawLimit := request.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > maxLimit {
			return pagination{}, fmt.Errorf(
				"limit must be an integer from 1 to %d",
				maxLimit,
			)
		}

		result.limit = limit
	}

	if rawOffset := request.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return pagination{}, fmt.Errorf(
				"offset must be a non-negative integer",
			)
		}

		result.offset = offset
	}

	return result, nil
}

func parseUUIDPathValue(
	request *http.Request,
	name string,
) (string, error) {
	value := request.PathValue(name)

	if err := uuid.Validate(value); err != nil {
		return "", fmt.Errorf("%s must be a valid UUID", name)
	}

	return value, nil
}

func parseUserID(request *http.Request) (string, error) {
	value := request.Header.Get(userIDHeader)

	if err := uuid.Validate(value); err != nil {
		return "", fmt.Errorf("%s header must be a valid UUID", userIDHeader)
	}

	return value, nil
}

func parseIdempotencyKey(request *http.Request) (string, error) {
	value := request.Header.Get(idempotencyKeyHeader)

	if value == "" || len(value) > maxIdempotencyKeyLength {
		return "", fmt.Errorf(
			"%s header must contain from 1 to %d characters",
			idempotencyKeyHeader,
			maxIdempotencyKeyLength,
		)
	}

	return value, nil
}

func parseOrderStatusFilter(
	request *http.Request,
	allowed func(domain.OrderStatus) bool,
) (*domain.OrderStatus, error) {
	raw := request.URL.Query().Get("status")
	if raw == "" {
		return nil, nil
	}

	status := domain.OrderStatus(raw)

	if !allowed(status) {
		return nil, fmt.Errorf("status %q is not supported", raw)
	}

	return &status, nil
}
