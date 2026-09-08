package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

func TestOrderHandlerCancel(t *testing.T) {
	var receivedOrderID, receivedUserID string

	handler := NewOrderHandler(fakeOrderService{
		cancel: func(
			_ context.Context,
			orderID string,
			userID string,
		) (domain.Order, error) {
			receivedOrderID = orderID
			receivedUserID = userID

			order := createdOrder()
			order.Status = domain.OrderStatusCancelled

			return order, nil
		},
	}, testLogger())

	request := newOrderRequest(
		t,
		http.MethodPost,
		"/api/v1/orders/"+testOrderID+"/cancel",
		"",
	)
	request.SetPathValue("orderId", testOrderID)

	recorder := httptest.NewRecorder()

	handler.Cancel(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d (%s)",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if receivedOrderID != testOrderID {
		t.Errorf("unexpected order id: %q", receivedOrderID)
	}

	if receivedUserID != testUserID {
		t.Errorf("unexpected user id: %q", receivedUserID)
	}

	var response orderResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != domain.OrderStatusCancelled {
		t.Errorf("unexpected status: %q", response.Status)
	}
}

func TestOrderHandlerCancelMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       string
	}{
		{
			name:       "already accepted by the restaurant",
			err:        domain.ErrOrderStatusForbidden,
			statusCode: http.StatusConflict,
			code:       "order_status_forbidden",
		},
		{
			name:       "foreign or unknown order",
			err:        domain.ErrOrderNotFound,
			statusCode: http.StatusNotFound,
			code:       "order_not_found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewOrderHandler(fakeOrderService{
				cancel: func(
					_ context.Context,
					_ string,
					_ string,
				) (domain.Order, error) {
					return domain.Order{}, test.err
				},
			}, testLogger())

			request := newOrderRequest(
				t,
				http.MethodPost,
				"/api/v1/orders/"+testOrderID+"/cancel",
				"",
			)
			request.SetPathValue("orderId", testOrderID)

			recorder := httptest.NewRecorder()

			handler.Cancel(recorder, request)

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

func TestOrderHandlerCancelRequiresUser(t *testing.T) {
	handler := NewOrderHandler(fakeOrderService{
		cancel: func(
			_ context.Context,
			_ string,
			_ string,
		) (domain.Order, error) {
			t.Fatal("service must not be called")

			return domain.Order{}, nil
		},
	}, testLogger())

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/orders/"+testOrderID+"/cancel",
		nil,
	)
	request.SetPathValue("orderId", testOrderID)

	recorder := httptest.NewRecorder()

	handler.Cancel(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}
