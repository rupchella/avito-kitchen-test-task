package httpserver

import (
	"net/http"
	"time"
)

type RouterDependencies struct {
	Pinger           Pinger
	ReadinessTimeout time.Duration
	OpenAPISpec      []byte
	Restaurants      *RestaurantHandler
	Orders           *OrderHandler
	Partner          *PartnerHandler
}

func NewRouter(dependencies RouterDependencies) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", Liveness)
	mux.HandleFunc(
		"GET /readyz",
		Readiness(dependencies.Pinger, dependencies.ReadinessTimeout),
	)

	if len(dependencies.OpenAPISpec) > 0 {
		mux.HandleFunc("GET /docs", SwaggerUI)
		mux.HandleFunc(
			"GET /openapi.yaml",
			OpenAPISpec(dependencies.OpenAPISpec),
		)
	}

	mux.HandleFunc(
		"GET /api/v1/restaurants",
		dependencies.Restaurants.List,
	)
	mux.HandleFunc(
		"GET /api/v1/restaurants/{restaurantId}",
		dependencies.Restaurants.Get,
	)
	mux.HandleFunc(
		"GET /api/v1/restaurants/{restaurantId}/menu",
		dependencies.Restaurants.Menu,
	)

	mux.HandleFunc(
		"GET /api/v1/orders",
		dependencies.Orders.List,
	)
	mux.HandleFunc(
		"POST /api/v1/orders",
		dependencies.Orders.Create,
	)
	mux.HandleFunc(
		"GET /api/v1/orders/{orderId}",
		dependencies.Orders.Get,
	)

	partner := dependencies.Partner

	mux.Handle(
		"PUT /api/v1/partner/menu",
		partner.Authenticated(partner.SyncMenu),
	)
	mux.Handle(
		"PATCH /api/v1/partner/menu-items/{externalId}/availability",
		partner.Authenticated(partner.SetMenuItemAvailability),
	)
	mux.Handle(
		"GET /api/v1/partner/orders",
		partner.Authenticated(partner.ListOrders),
	)
	mux.Handle(
		"PATCH /api/v1/partner/orders/{orderId}/status",
		partner.Authenticated(partner.UpdateOrderStatus),
	)

	return mux
}
