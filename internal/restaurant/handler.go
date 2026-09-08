package restaurant

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Handler struct {
	client *Client
	menu   []MenuItemInput
	logger *slog.Logger
}

func NewHandler(
	client *Client,
	menu []MenuItemInput,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		client: client,
		menu:   menu,
		logger: logger,
	}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /api/v1/menu", h.getMenu)
	mux.HandleFunc(
		"PATCH /api/v1/menu-items/{externalId}/availability",
		h.setAvailability,
	)

	return mux
}

func (h *Handler) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) getMenu(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"items": h.menu})
}

func (h *Handler) setAvailability(
	writer http.ResponseWriter,
	request *http.Request,
) {
	externalID := request.PathValue("externalId")

	var body struct {
		IsAvailable *bool `json:"is_available"`
	}

	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"code":    "validation_error",
			"message": "request body is not a valid JSON document",
		})

		return
	}

	if body.IsAvailable == nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"code":    "validation_error",
			"message": "is_available is required",
		})

		return
	}

	item, err := h.client.SetMenuItemAvailability(
		request.Context(),
		externalID,
		*body.IsAvailable,
	)
	if err != nil {
		h.logger.Error("failed to update availability", "error", err)

		var apiError *APIError

		if errors.As(err, &apiError) {
			writeJSON(writer, apiError.StatusCode, map[string]string{
				"code":    apiError.Code,
				"message": apiError.Message,
			})

			return
		}

		writeJSON(writer, http.StatusBadGateway, map[string]string{
			"code":    "kitchen_unavailable",
			"message": "kitchen API is unavailable",
		})

		return
	}

	writeJSON(writer, http.StatusOK, item)
}

func writeJSON(
	writer http.ResponseWriter,
	statusCode int,
	response any,
) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)

	_ = json.NewEncoder(writer).Encode(response)
}
