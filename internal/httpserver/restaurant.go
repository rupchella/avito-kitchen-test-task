package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type RestaurantService interface {
	ListActive(
		ctx context.Context,
		limit int,
		offset int,
	) ([]domain.Restaurant, int, error)

	Get(
		ctx context.Context,
		id string,
	) (domain.Restaurant, error)

	Menu(
		ctx context.Context,
		restaurantID string,
		search string,
		limit int,
		offset int,
	) (domain.Restaurant, []domain.MenuItem, int, error)

	SearchMenuItems(
		ctx context.Context,
		search string,
		limit int,
		offset int,
	) ([]domain.MenuSearchItem, int, error)
}

type RestaurantHandler struct {
	service RestaurantService
	logger  *slog.Logger
}

func NewRestaurantHandler(
	service RestaurantService,
	logger *slog.Logger,
) *RestaurantHandler {
	return &RestaurantHandler{
		service: service,
		logger:  logger,
	}
}

func (h *RestaurantHandler) List(
	writer http.ResponseWriter,
	request *http.Request,
) {
	page, err := parsePagination(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	restaurants, total, err := h.service.ListActive(
		request.Context(),
		page.limit,
		page.offset,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to list restaurants", err)

		return
	}

	writeJSON(writer, http.StatusOK, restaurantListResponse{
		Items:  newRestaurantResponses(restaurants),
		Total:  total,
		Limit:  page.limit,
		Offset: page.offset,
	})
}

func (h *RestaurantHandler) Get(
	writer http.ResponseWriter,
	request *http.Request,
) {
	restaurantID, err := parseUUIDPathValue(request, "restaurantId")
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	restaurant, err := h.service.Get(request.Context(), restaurantID)
	if err != nil {
		writeError(writer, h.logger, "failed to get restaurant", err)

		return
	}

	writeJSON(writer, http.StatusOK, newRestaurantResponse(restaurant))
}

func (h *RestaurantHandler) Menu(
	writer http.ResponseWriter,
	request *http.Request,
) {
	restaurantID, err := parseUUIDPathValue(request, "restaurantId")
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	search, err := parseSearch(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	page, err := parsePagination(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	restaurant, items, total, err := h.service.Menu(
		request.Context(),
		restaurantID,
		search,
		page.limit,
		page.offset,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to get restaurant menu", err)

		return
	}

	writeJSON(writer, http.StatusOK, menuResponse{
		Restaurant: newRestaurantResponse(restaurant),
		Items:      newMenuItemResponses(items),
		Total:      total,
		Limit:      page.limit,
		Offset:     page.offset,
	})
}

// SearchMenuItems отдаёт блюда со всей площадки: пользователь ищет
// товар, а не заведение.
func (h *RestaurantHandler) SearchMenuItems(
	writer http.ResponseWriter,
	request *http.Request,
) {
	search, err := parseSearch(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	page, err := parsePagination(request)
	if err != nil {
		writeValidationError(writer, err.Error())

		return
	}

	items, total, err := h.service.SearchMenuItems(
		request.Context(),
		search,
		page.limit,
		page.offset,
	)
	if err != nil {
		writeError(writer, h.logger, "failed to search menu items", err)

		return
	}

	found := make([]menuSearchItemResponse, 0, len(items))

	for _, item := range items {
		found = append(found, menuSearchItemResponse{
			menuItemResponse: newMenuItemResponse(item.MenuItem),
			Restaurant: restaurantSummaryResponse{
				ID:   item.RestaurantID,
				Name: item.RestaurantName,
			},
		})
	}

	writeJSON(writer, http.StatusOK, menuSearchResponse{
		Items:  found,
		Total:  total,
		Limit:  page.limit,
		Offset: page.offset,
	})
}

type restaurantResponse struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Status    domain.RestaurantStatus `json:"status"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type restaurantListResponse struct {
	Items  []restaurantResponse `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

type menuItemResponse struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	ExternalID   string    `json:"external_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	PriceKopecks int64     `json:"price_kopecks"`
	IsAvailable  bool      `json:"is_available"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type restaurantSummaryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type menuSearchItemResponse struct {
	menuItemResponse

	Restaurant restaurantSummaryResponse `json:"restaurant"`
}

type menuSearchResponse struct {
	Items  []menuSearchItemResponse `json:"items"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

type menuResponse struct {
	Restaurant restaurantResponse `json:"restaurant"`
	Items      []menuItemResponse `json:"items"`
	Total      int                `json:"total"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

func newRestaurantResponse(
	restaurant domain.Restaurant,
) restaurantResponse {
	return restaurantResponse{
		ID:        restaurant.ID,
		Name:      restaurant.Name,
		Status:    restaurant.Status,
		CreatedAt: restaurant.CreatedAt,
		UpdatedAt: restaurant.UpdatedAt,
	}
}

func newRestaurantResponses(
	restaurants []domain.Restaurant,
) []restaurantResponse {
	responses := make([]restaurantResponse, 0, len(restaurants))

	for _, restaurant := range restaurants {
		responses = append(responses, newRestaurantResponse(restaurant))
	}

	return responses
}

func newMenuItemResponse(item domain.MenuItem) menuItemResponse {
	return menuItemResponse{
		ID:           item.ID,
		RestaurantID: item.RestaurantID,
		ExternalID:   item.ExternalID,
		Name:         item.Name,
		Description:  item.Description,
		PriceKopecks: item.PriceKopecks,
		IsAvailable:  item.IsAvailable,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func newMenuItemResponses(items []domain.MenuItem) []menuItemResponse {
	responses := make([]menuItemResponse, 0, len(items))

	for _, item := range items {
		responses = append(responses, newMenuItemResponse(item))
	}

	return responses
}
