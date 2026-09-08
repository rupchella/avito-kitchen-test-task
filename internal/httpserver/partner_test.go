package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const testAPIKey = "demo-restaurant-key"

func newPartnerRequest(
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
	request.Header.Set(apiKeyHeader, testAPIKey)

	return request
}

func authenticatingService() fakePartnerService {
	return fakePartnerService{
		authenticate: func(
			_ context.Context,
			apiKey string,
		) (string, error) {
			if apiKey != testAPIKey {
				return "", domain.ErrPartnerUnauthorized
			}

			return testRestaurantID, nil
		},
	}
}

func TestPartnerHandlerRejectsUnknownAPIKey(t *testing.T) {
	handler := NewPartnerHandler(authenticatingService(), testLogger())

	called := false

	authenticated := handler.Authenticated(func(
		_ http.ResponseWriter,
		_ *http.Request,
		_ string,
	) {
		called = true
	})

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/partner/orders",
		nil,
	)
	request.Header.Set(apiKeyHeader, "wrong-key")

	recorder := httptest.NewRecorder()

	authenticated(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			recorder.Code,
		)
	}

	if called {
		t.Error("handler must not be called without a valid API key")
	}

	var response errorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Code != "unauthorized" {
		t.Errorf("unexpected error code: %q", response.Code)
	}
}

func TestPartnerHandlerPassesRestaurantToHandler(t *testing.T) {
	handler := NewPartnerHandler(authenticatingService(), testLogger())

	var received string

	authenticated := handler.Authenticated(func(
		writer http.ResponseWriter,
		_ *http.Request,
		restaurantID string,
	) {
		received = restaurantID

		writer.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()

	authenticated(
		recorder,
		newPartnerRequest(t, http.MethodGet, "/api/v1/partner/orders", ""),
	)

	if received != testRestaurantID {
		t.Errorf("unexpected restaurant id: %q", received)
	}
}

func TestPartnerHandlerSyncMenu(t *testing.T) {
	var received []domain.MenuItemInput

	partner := authenticatingService()
	partner.syncMenu = func(
		_ context.Context,
		_ string,
		inputs []domain.MenuItemInput,
	) ([]domain.MenuItem, error) {
		received = inputs

		return []domain.MenuItem{
			{
				ID:           testMenuItemID,
				RestaurantID: testRestaurantID,
				ExternalID:   inputs[0].ExternalID,
				Name:         inputs[0].Name,
				PriceKopecks: inputs[0].PriceKopecks,
				IsAvailable:  inputs[0].IsAvailable,
			},
		}, nil
	}

	handler := NewPartnerHandler(partner, testLogger())

	body := `{"items":[{"external_id":"pizza-margherita",` +
		`"name":"Пицца Маргарита","description":"Томаты и моцарелла",` +
		`"price_kopecks":49900,"is_available":true}]}`

	recorder := httptest.NewRecorder()

	handler.SyncMenu(
		recorder,
		newPartnerRequest(t, http.MethodPut, "/api/v1/partner/menu", body),
		testRestaurantID,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d (%s)",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 menu item, got %d", len(received))
	}

	if received[0].ExternalID != "pizza-margherita" {
		t.Errorf("unexpected external id: %q", received[0].ExternalID)
	}

	if received[0].PriceKopecks != 49900 {
		t.Errorf("unexpected price: %d", received[0].PriceKopecks)
	}

	var response menuSyncResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.RestaurantID != testRestaurantID {
		t.Errorf("unexpected restaurant id: %q", response.RestaurantID)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item in response, got %d", len(response.Items))
	}
}

func TestPartnerHandlerSyncMenuRejectsInvalidItems(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty external id",
			body: `{"items":[{"external_id":"","name":"Пицца","price_kopecks":100,"is_available":true}]}`,
		},
		{
			name: "empty name",
			body: `{"items":[{"external_id":"pizza","name":"","price_kopecks":100,"is_available":true}]}`,
		},
		{
			name: "zero price",
			body: `{"items":[{"external_id":"pizza","name":"Пицца","price_kopecks":0,"is_available":true}]}`,
		},
		{
			name: "negative price",
			body: `{"items":[{"external_id":"pizza","name":"Пицца","price_kopecks":-100,"is_available":true}]}`,
		},
		{
			name: "missing availability",
			body: `{"items":[{"external_id":"pizza","name":"Пицца","price_kopecks":100}]}`,
		},
		{
			name: "unknown field",
			body: `{"items":[{"external_id":"pizza","name":"Пицца","price_kopecks":100,"is_available":true,"weight":300}]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			partner := authenticatingService()
			partner.syncMenu = func(
				_ context.Context,
				_ string,
				_ []domain.MenuItemInput,
			) ([]domain.MenuItem, error) {
				t.Fatal("service must not be called")

				return nil, nil
			}

			handler := NewPartnerHandler(partner, testLogger())
			recorder := httptest.NewRecorder()

			handler.SyncMenu(
				recorder,
				newPartnerRequest(
					t,
					http.MethodPut,
					"/api/v1/partner/menu",
					test.body,
				),
				testRestaurantID,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					recorder.Code,
				)
			}
		})
	}
}

func TestPartnerHandlerSetAvailability(t *testing.T) {
	partner := authenticatingService()

	var receivedExternalID string
	var receivedAvailability bool

	partner.setAvailability = func(
		_ context.Context,
		_ string,
		externalID string,
		isAvailable bool,
	) (domain.MenuItem, error) {
		receivedExternalID = externalID
		receivedAvailability = isAvailable

		return domain.MenuItem{
			ID:          testMenuItemID,
			ExternalID:  externalID,
			IsAvailable: isAvailable,
		}, nil
	}

	handler := NewPartnerHandler(partner, testLogger())

	request := newPartnerRequest(
		t,
		http.MethodPatch,
		"/api/v1/partner/menu-items/pizza-margherita/availability",
		`{"is_available":false}`,
	)
	request.SetPathValue("externalId", "pizza-margherita")

	recorder := httptest.NewRecorder()

	handler.SetMenuItemAvailability(recorder, request, testRestaurantID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if receivedExternalID != "pizza-margherita" {
		t.Errorf("unexpected external id: %q", receivedExternalID)
	}

	if receivedAvailability {
		t.Error("expected is_available to be false")
	}
}

func TestPartnerHandlerSetAvailabilityRequiresFlag(t *testing.T) {
	partner := authenticatingService()
	partner.setAvailability = func(
		_ context.Context,
		_ string,
		_ string,
		_ bool,
	) (domain.MenuItem, error) {
		t.Fatal("service must not be called")

		return domain.MenuItem{}, nil
	}

	handler := NewPartnerHandler(partner, testLogger())

	request := newPartnerRequest(
		t,
		http.MethodPatch,
		"/api/v1/partner/menu-items/pizza-margherita/availability",
		`{}`,
	)
	request.SetPathValue("externalId", "pizza-margherita")

	recorder := httptest.NewRecorder()

	handler.SetMenuItemAvailability(recorder, request, testRestaurantID)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestPartnerHandlerListOrdersUsesCreatedByDefault(t *testing.T) {
	partner := authenticatingService()

	var received *domain.OrderStatus

	partner.listOrders = func(
		_ context.Context,
		_ string,
		status *domain.OrderStatus,
		_ int,
		_ int,
	) ([]domain.Order, int, error) {
		received = status

		return []domain.Order{}, 0, nil
	}

	handler := NewPartnerHandler(partner, testLogger())
	recorder := httptest.NewRecorder()

	handler.ListOrders(
		recorder,
		newPartnerRequest(t, http.MethodGet, "/api/v1/partner/orders", ""),
		testRestaurantID,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if received == nil || *received != domain.OrderStatusCreated {
		t.Fatalf("expected created filter, got %v", received)
	}
}

func TestPartnerHandlerUpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		err        error
		statusCode int
		code       string
	}{
		{
			name:       "accepted",
			body:       `{"status":"accepted"}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "created is not allowed for a restaurant",
			body:       `{"status":"created"}`,
			statusCode: http.StatusBadRequest,
			code:       "validation_error",
		},
		{
			name:       "unknown status",
			body:       `{"status":"delivered"}`,
			statusCode: http.StatusBadRequest,
			code:       "validation_error",
		},
		{
			name:       "forbidden transition",
			body:       `{"status":"ready"}`,
			err:        domain.ErrOrderStatusForbidden,
			statusCode: http.StatusConflict,
			code:       "order_status_forbidden",
		},
		{
			name:       "foreign order",
			body:       `{"status":"accepted"}`,
			err:        domain.ErrOrderNotFound,
			statusCode: http.StatusNotFound,
			code:       "order_not_found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			partner := authenticatingService()
			partner.updateOrderStatus = func(
				_ context.Context,
				_ string,
				_ string,
				target domain.OrderStatus,
			) (domain.Order, error) {
				if test.err != nil {
					return domain.Order{}, test.err
				}

				order := createdOrder()
				order.Status = target

				return order, nil
			}

			handler := NewPartnerHandler(partner, testLogger())

			request := newPartnerRequest(
				t,
				http.MethodPatch,
				"/api/v1/partner/orders/"+testOrderID+"/status",
				test.body,
			)
			request.SetPathValue("orderId", testOrderID)

			recorder := httptest.NewRecorder()

			handler.UpdateOrderStatus(recorder, request, testRestaurantID)

			if recorder.Code != test.statusCode {
				t.Fatalf(
					"expected status %d, got %d (%s)",
					test.statusCode,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if test.code == "" {
				return
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
