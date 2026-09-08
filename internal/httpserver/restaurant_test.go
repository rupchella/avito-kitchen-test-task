package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

func TestRestaurantHandlerList(t *testing.T) {
	createdAt := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.September, 8, 13, 0, 0, 0, time.UTC)

	lister := fakeRestaurantService{
		listActive: func(
			_ context.Context,
			limit int,
			offset int,
		) ([]domain.Restaurant, int, error) {
			if limit != 10 {
				t.Fatalf("expected limit 10, got %d", limit)
			}

			if offset != 2 {
				t.Fatalf("expected offset 2, got %d", offset)
			}

			return []domain.Restaurant{
				{
					ID:        "11111111-1111-4111-8111-111111111111",
					Name:      "Пицца у Лены",
					Status:    domain.RestaurantStatusActive,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			}, 1, nil
		},
	}

	handler := NewRestaurantHandler(
		lister,
		testLogger(),
	)

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/restaurants?limit=10&offset=2",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"expected application/json content type, got %q",
			contentType,
		)
	}

	var response restaurantListResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Total != 1 {
		t.Fatalf("expected total 1, got %d", response.Total)
	}

	if response.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", response.Limit)
	}

	if response.Offset != 2 {
		t.Fatalf("expected offset 2, got %d", response.Offset)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 restaurant, got %d", len(response.Items))
	}

	restaurant := response.Items[0]

	if restaurant.ID != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("unexpected restaurant ID: %q", restaurant.ID)
	}

	if restaurant.Name != "Пицца у Лены" {
		t.Errorf("unexpected restaurant name: %q", restaurant.Name)
	}

	if restaurant.Status != domain.RestaurantStatusActive {
		t.Errorf("unexpected restaurant status: %q", restaurant.Status)
	}

	if !restaurant.CreatedAt.Equal(createdAt) {
		t.Errorf("unexpected created_at: %s", restaurant.CreatedAt)
	}

	if !restaurant.UpdatedAt.Equal(updatedAt) {
		t.Errorf("unexpected updated_at: %s", restaurant.UpdatedAt)
	}
}

func TestRestaurantHandlerListRejectsInvalidPagination(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "limit is not an integer",
			target: "/api/v1/restaurants?limit=abc",
		},
		{
			name:   "limit is greater than maximum",
			target: "/api/v1/restaurants?limit=101",
		},
		{
			name:   "offset is not an integer",
			target: "/api/v1/restaurants?offset=abc",
		},
		{
			name:   "offset is negative",
			target: "/api/v1/restaurants?offset=-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceCalled := false

			lister := fakeRestaurantService{
				listActive: func(
					_ context.Context,
					_ int,
					_ int,
				) ([]domain.Restaurant, int, error) {
					serviceCalled = true
					return nil, 0, nil
				},
			}

			handler := NewRestaurantHandler(
				lister,
				testLogger(),
			)

			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				test.target,
				nil,
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

			if serviceCalled {
				t.Fatal("service must not be called for invalid pagination")
			}

			var response errorResponse

			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if response.Code != "validation_error" {
				t.Errorf(
					"expected validation_error, got %q",
					response.Code,
				)
			}
		})
	}
}

func TestRestaurantHandlerListReturnsInternalError(t *testing.T) {
	lister := fakeRestaurantService{
		listActive: func(
			_ context.Context,
			_ int,
			_ int,
		) ([]domain.Restaurant, int, error) {
			return nil, 0, errors.New("database failure")
		},
	}

	handler := NewRestaurantHandler(
		lister,
		testLogger(),
	)

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/restaurants",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	var response errorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Code != "internal_error" {
		t.Errorf(
			"expected internal_error, got %q",
			response.Code,
		)
	}
}
