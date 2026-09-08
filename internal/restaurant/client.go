package restaurant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(
	baseURL string,
	apiKey string,
	httpClient *http.Client,
) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

type MenuItemInput struct {
	ExternalID   string `json:"external_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceKopecks int64  `json:"price_kopecks"`
	IsAvailable  bool   `json:"is_available"`
}

type MenuItem struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	ExternalID   string `json:"external_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceKopecks int64  `json:"price_kopecks"`
	IsAvailable  bool   `json:"is_available"`
}

type Order struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	TotalPriceKopecks int64  `json:"total_price_kopecks"`
	DeliveryAddress   string `json:"delivery_address"`
}

type orderListResponse struct {
	Items []Order `json:"items"`
	Total int     `json:"total"`
}

type menuSyncResponse struct {
	RestaurantID string     `json:"restaurant_id"`
	Items        []MenuItem `json:"items"`
}

// APIError описывает неуспешный ответ партнёрского API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"kitchen API responded with %d: %s (%s)",
		e.StatusCode,
		e.Message,
		e.Code,
	)
}

func (c *Client) SyncMenu(
	ctx context.Context,
	items []MenuItemInput,
) ([]MenuItem, error) {
	body := struct {
		Items []MenuItemInput `json:"items"`
	}{
		Items: items,
	}

	var response menuSyncResponse

	if err := c.do(
		ctx,
		http.MethodPut,
		"/api/v1/partner/menu",
		body,
		&response,
	); err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (c *Client) ListOrders(
	ctx context.Context,
	status string,
	limit int,
) ([]Order, error) {
	query := url.Values{}
	query.Set("status", status)
	query.Set("limit", strconv.Itoa(limit))

	var response orderListResponse

	if err := c.do(
		ctx,
		http.MethodGet,
		"/api/v1/partner/orders?"+query.Encode(),
		nil,
		&response,
	); err != nil {
		return nil, err
	}

	return response.Items, nil
}

func (c *Client) UpdateOrderStatus(
	ctx context.Context,
	orderID string,
	status string,
) (Order, error) {
	body := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	var order Order

	if err := c.do(
		ctx,
		http.MethodPatch,
		"/api/v1/partner/orders/"+url.PathEscape(orderID)+"/status",
		body,
		&order,
	); err != nil {
		return Order{}, err
	}

	return order, nil
}

func (c *Client) SetMenuItemAvailability(
	ctx context.Context,
	externalID string,
	isAvailable bool,
) (MenuItem, error) {
	body := struct {
		IsAvailable bool `json:"is_available"`
	}{
		IsAvailable: isAvailable,
	}

	var item MenuItem

	path := "/api/v1/partner/menu-items/" +
		url.PathEscape(externalID) +
		"/availability"

	if err := c.do(ctx, http.MethodPatch, path, body, &item); err != nil {
		return MenuItem{}, err
	}

	return item, nil
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	requestBody any,
	responseBody any,
) error {
	var reader io.Reader

	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}

		reader = bytes.NewReader(encoded)
	}

	// Адрес платформы задаётся конфигурацией сервиса, а не запросом.
	request, err := http.NewRequestWithContext( //nolint:gosec // адрес из конфигурации
		ctx,
		method,
		c.baseURL+path,
		reader,
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("X-API-Key", c.apiKey)

	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request) //nolint:gosec // адрес из конфигурации
	if err != nil {
		return fmt.Errorf("call kitchen API: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode >= http.StatusBadRequest {
		return newAPIError(response)
	}

	if responseBody == nil {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func newAPIError(response *http.Response) error {
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	_ = json.NewDecoder(response.Body).Decode(&payload)

	return &APIError{
		StatusCode: response.StatusCode,
		Code:       payload.Code,
		Message:    payload.Message,
	}
}
