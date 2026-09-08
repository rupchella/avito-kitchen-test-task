package domain

import "errors"

var (
	ErrRestaurantNotFound = errors.New("restaurant not found")

	ErrMenuItemNotFound     = errors.New("menu item not found")
	ErrMenuItemUnavailable  = errors.New("menu item is not available")
	ErrDuplicateMenuItem    = errors.New("menu item is duplicated in request")
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderStatusForbidden = errors.New("order status transition is not allowed")

	ErrIdempotencyConflict = errors.New("idempotency key was used for another order")

	ErrPartnerUnauthorized = errors.New("partner API key is invalid")
)
