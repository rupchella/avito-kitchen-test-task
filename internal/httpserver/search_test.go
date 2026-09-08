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

func TestRestaurantHandlerSearchMenuItems(t *testing.T) {
	var receivedSearch string

	handler := NewRestaurantHandler(fakeRestaurantService{
		searchMenuItems: func(
			_ context.Context,
			search string,
			_ int,
			_ int,
		) ([]domain.MenuSearchItem, int, error) {
			receivedSearch = search

			return []domain.MenuSearchItem{
				{
					MenuItem: domain.MenuItem{
						ID:           testMenuItemID,
						RestaurantID: testRestaurantID,
						ExternalID:   "pizza-margherita",
						Name:         "Пицца Маргарита",
						PriceKopecks: 49900,
						IsAvailable:  true,
					},
					RestaurantName: "AvitoCookRestaurant",
				},
			}, 1, nil
		},
	}, testLogger())

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/menu-items?search=%D0%BF%D0%B8%D1%86%D1%86%D0%B0",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.SearchMenuItems(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d (%s)",
			http.StatusOK,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if receivedSearch != "пицца" {
		t.Errorf("unexpected search query: %q", receivedSearch)
	}

	var response menuSearchResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(response.Items))
	}

	item := response.Items[0]

	if item.Restaurant.Name != "AvitoCookRestaurant" {
		t.Errorf("unexpected restaurant name: %q", item.Restaurant.Name)
	}

	if item.Restaurant.ID != testRestaurantID {
		t.Errorf("unexpected restaurant id: %q", item.Restaurant.ID)
	}

	if item.Name != "Пицца Маргарита" {
		t.Errorf("unexpected item name: %q", item.Name)
	}
}

func TestRestaurantHandlerSearchTrimsAndAllowsEmptyQuery(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "no query",
			target:   "/api/v1/menu-items",
			expected: "",
		},
		{
			name:     "blank query",
			target:   "/api/v1/menu-items?search=%20%20",
			expected: "",
		},
		{
			name:     "trimmed query",
			target:   "/api/v1/menu-items?search=%20pizza%20",
			expected: "pizza",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var receivedSearch string

			handler := NewRestaurantHandler(fakeRestaurantService{
				searchMenuItems: func(
					_ context.Context,
					search string,
					_ int,
					_ int,
				) ([]domain.MenuSearchItem, int, error) {
					receivedSearch = search

					return nil, 0, nil
				},
			}, testLogger())

			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				test.target,
				nil,
			)
			recorder := httptest.NewRecorder()

			handler.SearchMenuItems(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}

			if receivedSearch != test.expected {
				t.Errorf(
					"expected search %q, got %q",
					test.expected,
					receivedSearch,
				)
			}
		})
	}
}

func TestRestaurantHandlerSearchRejectsTooLongQuery(t *testing.T) {
	handler := NewRestaurantHandler(fakeRestaurantService{
		searchMenuItems: func(
			_ context.Context,
			_ string,
			_ int,
			_ int,
		) ([]domain.MenuSearchItem, int, error) {
			t.Fatal("service must not be called")

			return nil, 0, nil
		},
	}, testLogger())

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/menu-items?search="+strings.Repeat("a", maxSearchLength+1),
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.SearchMenuItems(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestRestaurantHandlerMenuPassesSearch(t *testing.T) {
	var receivedSearch string

	handler := NewRestaurantHandler(fakeRestaurantService{
		menu: func(
			_ context.Context,
			_ string,
			search string,
			_ int,
			_ int,
		) (domain.Restaurant, []domain.MenuItem, int, error) {
			receivedSearch = search

			return domain.Restaurant{ID: testRestaurantID}, nil, 0, nil
		},
	}, testLogger())

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/restaurants/"+testRestaurantID+"/menu?search=pizza",
		nil,
	)
	request.SetPathValue("restaurantId", testRestaurantID)

	recorder := httptest.NewRecorder()

	handler.Menu(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if receivedSearch != "pizza" {
		t.Errorf("unexpected search query: %q", receivedSearch)
	}
}
