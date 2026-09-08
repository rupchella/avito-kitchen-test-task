package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/service"
)

const (
	testUserID       = "11111111-1111-4111-8111-111111111111"
	testRestaurantID = "a1b2c3d4-0000-4000-8000-000000000001"
	testMenuItemID   = "22222222-2222-4222-8222-222222222222"
	testOrderID      = "33333333-3333-4333-8333-333333333333"
)

const validOrderBody = `{
	"restaurant_id": "a1b2c3d4-0000-4000-8000-000000000001",
	"delivery_address": "Батуми, улица Пушкина, дом 10",
	"items": [
		{
			"menu_item_id": "22222222-2222-4222-8222-222222222222",
			"quantity": 2
		}
	]
}`

func createdOrder() domain.Order {
	return domain.Order{
		ID:                testOrderID,
		RestaurantID:      testRestaurantID,
		UserID:            testUserID,
		Status:            domain.OrderStatusCreated,
		TotalPriceKopecks: 99800,
		DeliveryAddress:   "Батуми, улица Пушкина, дом 10",
		Items: []domain.OrderItem{
			{
				MenuItemID:       testMenuItemID,
				ItemName:         "Пицца Маргарита",
				UnitPriceKopecks: 49900,
				Quantity:         2,
			},
		},
	}
}

func newOrderRequest(
	t *testing.T,
	method string,
	target string,
	body string,
) *http.Request {
	t.Helper()

	request := httptest.NewRequestWithContext(
		t.Context(),
		method,
		target,
		strings.NewReader(body),
	)
	request.Header.Set(userIDHeader, testUserID)
	request.Header.Set(idempotencyKeyHeader, "key-1")

	return request
}

func TestOrderHandlerCreate(t *testing.T) {
	var received service.CreateOrderInput

	handler := NewOrderHandler(fakeOrderService{
		create: func(
			_ context.Context,
			input service.CreateOrderInput,
		) (domain.Order, bool, error) {
			received = input

			return createdOrder(), true, nil
		},
	}, testLogger())

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		newOrderRequest(t, http.MethodPost, "/api/v1/orders", validOrderBody),
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d (%s)",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if location := recorder.Header().Get("Location"); location != "/api/v1/orders/"+testOrderID {
		t.Errorf("unexpected Location header: %q", location)
	}

	if received.UserID != testUserID {
		t.Errorf("unexpected user id: %q", received.UserID)
	}

	if received.IdempotencyKey != "key-1" {
		t.Errorf("unexpected idempotency key: %q", received.IdempotencyKey)
	}

	if len(received.Items) != 1 || received.Items[0].Quantity != 2 {
		t.Fatalf("unexpected items: %+v", received.Items)
	}

	var response orderResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.TotalPriceKopecks != 99800 {
		t.Errorf("unexpected total price: %d", response.TotalPriceKopecks)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 position, got %d", len(response.Items))
	}

	if response.Items[0].TotalPriceKopecks != 99800 {
		t.Errorf(
			"unexpected position total: %d",
			response.Items[0].TotalPriceKopecks,
		)
	}
}

func TestOrderHandlerCreateReturnsExistingOrder(t *testing.T) {
	handler := NewOrderHandler(fakeOrderService{
		create: func(
			_ context.Context,
			_ service.CreateOrderInput,
		) (domain.Order, bool, error) {
			return createdOrder(), false, nil
		},
	}, testLogger())

	recorder := httptest.NewRecorder()

	handler.Create(
		recorder,
		newOrderRequest(t, http.MethodPost, "/api/v1/orders", validOrderBody),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if location := recorder.Header().Get("Location"); location != "" {
		t.Errorf("repeated request must not set Location: %q", location)
	}
}

func TestOrderHandlerCreateRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		key     string
		body    string
		message string
	}{
		{
			name:    "missing user id",
			userID:  "",
			key:     "key-1",
			body:    validOrderBody,
			message: "X-User-ID header must be a valid UUID",
		},
		{
			name:    "user id is not a uuid",
			userID:  "user-1",
			key:     "key-1",
			body:    validOrderBody,
			message: "X-User-ID header must be a valid UUID",
		},
		{
			name:   "missing idempotency key",
			userID: testUserID,
			key:    "",
			body:   validOrderBody,
			message: "Idempotency-Key header must contain from 1 to 128 " +
				"characters",
		},
		{
			name:    "malformed body",
			userID:  testUserID,
			key:     "key-1",
			body:    "{",
			message: "",
		},
		{
			name:   "unknown field",
			userID: testUserID,
			key:    "key-1",
			body: `{"restaurant_id":"a1b2c3d4-0000-4000-8000-000000000001",` +
				`"delivery_address":"Батуми, Пушкина 10",` +
				`"items":[{"menu_item_id":"22222222-2222-4222-8222-222222222222",` +
				`"quantity":1}],"comment":"без лука"}`,
			message: "",
		},
		{
			name:    "empty items",
			userID:  testUserID,
			key:     "key-1",
			body:    `{"restaurant_id":"a1b2c3d4-0000-4000-8000-000000000001","delivery_address":"Батуми, Пушкина 10","items":[]}`,
			message: "items must contain from 1 to 100 positions",
		},
		{
			name:    "zero quantity",
			userID:  testUserID,
			key:     "key-1",
			body:    `{"restaurant_id":"a1b2c3d4-0000-4000-8000-000000000001","delivery_address":"Батуми, Пушкина 10","items":[{"menu_item_id":"22222222-2222-4222-8222-222222222222","quantity":0}]}`,
			message: "quantity must be an integer from 1 to 100",
		},
		{
			name:    "short address",
			userID:  testUserID,
			key:     "key-1",
			body:    `{"restaurant_id":"a1b2c3d4-0000-4000-8000-000000000001","delivery_address":"дом","items":[{"menu_item_id":"22222222-2222-4222-8222-222222222222","quantity":1}]}`,
			message: "delivery_address must contain from 5 to 1000 characters",
		},
		{
			name:    "restaurant id is not a uuid",
			userID:  testUserID,
			key:     "key-1",
			body:    `{"restaurant_id":"pizza","delivery_address":"Батуми, Пушкина 10","items":[{"menu_item_id":"22222222-2222-4222-8222-222222222222","quantity":1}]}`,
			message: "restaurant_id must be a valid UUID",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewOrderHandler(fakeOrderService{
				create: func(
					_ context.Context,
					_ service.CreateOrderInput,
				) (domain.Order, bool, error) {
					t.Fatal("service must not be called")

					return domain.Order{}, false, nil
				},
			}, testLogger())

			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/api/v1/orders",
				strings.NewReader(test.body),
			)

			if test.userID != "" {
				request.Header.Set(userIDHeader, test.userID)
			}

			if test.key != "" {
				request.Header.Set(idempotencyKeyHeader, test.key)
			}

			recorder := httptest.NewRecorder()

			handler.Create(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					recorder.Code,
				)
			}

			var response errorResponse

			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if response.Code != "validation_error" {
				t.Errorf("unexpected error code: %q", response.Code)
			}

			if test.message != "" && response.Message != test.message {
				t.Errorf("unexpected message: %q", response.Message)
			}
		})
	}
}

func TestOrderHandlerCreateMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       string
	}{
		{
			name:       "unavailable item",
			err:        domain.ErrMenuItemUnavailable,
			statusCode: http.StatusConflict,
			code:       "menu_item_unavailable",
		},
		{
			name:       "unknown item",
			err:        domain.ErrMenuItemNotFound,
			statusCode: http.StatusNotFound,
			code:       "menu_item_not_found",
		},
		{
			name:       "unknown restaurant",
			err:        domain.ErrRestaurantNotFound,
			statusCode: http.StatusNotFound,
			code:       "restaurant_not_found",
		},
		{
			name:       "reused idempotency key",
			err:        domain.ErrIdempotencyConflict,
			statusCode: http.StatusConflict,
			code:       "idempotency_conflict",
		},
		{
			name:       "duplicated item",
			err:        domain.ErrDuplicateMenuItem,
			statusCode: http.StatusBadRequest,
			code:       "validation_error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewOrderHandler(fakeOrderService{
				create: func(
					_ context.Context,
					_ service.CreateOrderInput,
				) (domain.Order, bool, error) {
					return domain.Order{}, false, test.err
				},
			}, testLogger())

			recorder := httptest.NewRecorder()

			handler.Create(
				recorder,
				newOrderRequest(
					t,
					http.MethodPost,
					"/api/v1/orders",
					validOrderBody,
				),
			)

			if recorder.Code != test.statusCode {
				t.Fatalf(
					"expected status %d, got %d",
					test.statusCode,
					recorder.Code,
				)
			}

			var response errorResponse

			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if response.Code != test.code {
				t.Errorf("unexpected error code: %q", response.Code)
			}
		})
	}
}

func TestOrderHandlerGetReturnsNotFound(t *testing.T) {
	handler := NewOrderHandler(fakeOrderService{
		get: func(
			_ context.Context,
			_ string,
			_ string,
		) (domain.Order, error) {
			return domain.Order{}, domain.ErrOrderNotFound
		},
	}, testLogger())

	request := newOrderRequest(
		t,
		http.MethodGet,
		"/api/v1/orders/"+testOrderID,
		"",
	)
	request.SetPathValue("orderId", testOrderID)

	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func TestOrderHandlerListPassesStatusFilter(t *testing.T) {
	var received *domain.OrderStatus

	handler := NewOrderHandler(fakeOrderService{
		list: func(
			_ context.Context,
			_ string,
			status *domain.OrderStatus,
			_ int,
			_ int,
		) ([]domain.Order, int, error) {
			received = status

			return []domain.Order{createdOrder()}, 1, nil
		},
	}, testLogger())

	request := newOrderRequest(
		t,
		http.MethodGet,
		"/api/v1/orders?status=ready",
		"",
	)
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if received == nil || *received != domain.OrderStatusReady {
		t.Fatalf("unexpected status filter: %v", received)
	}
}

func TestOrderHandlerListRejectsUnknownStatus(t *testing.T) {
	handler := NewOrderHandler(fakeOrderService{
		list: func(
			_ context.Context,
			_ string,
			_ *domain.OrderStatus,
			_ int,
			_ int,
		) ([]domain.Order, int, error) {
			t.Fatal("service must not be called")

			return nil, 0, nil
		},
	}, testLogger())

	request := newOrderRequest(
		t,
		http.MethodGet,
		"/api/v1/orders?status=refunded",
		"",
	)
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}
