package domain

import "testing"

func TestOrderStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    OrderStatus
		to      OrderStatus
		allowed bool
	}{
		{"created to accepted", OrderStatusCreated, OrderStatusAccepted, true},
		{"created to rejected", OrderStatusCreated, OrderStatusRejected, true},
		{"created to preparing", OrderStatusCreated, OrderStatusPreparing, false},
		{"created to ready", OrderStatusCreated, OrderStatusReady, false},
		{"accepted to preparing", OrderStatusAccepted, OrderStatusPreparing, true},
		{"accepted to rejected", OrderStatusAccepted, OrderStatusRejected, false},
		{"preparing to ready", OrderStatusPreparing, OrderStatusReady, true},
		{"ready is terminal", OrderStatusReady, OrderStatusPreparing, false},
		{"rejected is terminal", OrderStatusRejected, OrderStatusAccepted, false},
		{"unknown status", OrderStatus("cooking"), OrderStatusReady, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.from.CanTransitionTo(test.to); got != test.allowed {
				t.Errorf(
					"transition %s -> %s: expected %t, got %t",
					test.from,
					test.to,
					test.allowed,
					got,
				)
			}
		})
	}
}

func TestOrderStatusIsValid(t *testing.T) {
	valid := []OrderStatus{
		OrderStatusCreated,
		OrderStatusAccepted,
		OrderStatusRejected,
		OrderStatusPreparing,
		OrderStatusReady,
	}

	for _, status := range valid {
		if !status.IsValid() {
			t.Errorf("status %q must be valid", status)
		}
	}

	if OrderStatus("delivered").IsValid() {
		t.Error("unknown status must not be valid")
	}
}

func TestOrderDraftTotalPriceKopecks(t *testing.T) {
	draft := OrderDraft{
		Items: []OrderItem{
			{UnitPriceKopecks: 49900, Quantity: 2},
			{UnitPriceKopecks: 19000, Quantity: 3},
		},
	}

	const expected = 49900*2 + 19000*3

	if total := draft.TotalPriceKopecks(); total != expected {
		t.Errorf("expected total %d, got %d", expected, total)
	}
}

func TestOrderDraftMatches(t *testing.T) {
	order := Order{
		RestaurantID:    "restaurant-1",
		DeliveryAddress: "Батуми, улица Пушкина, дом 10",
		Items: []OrderItem{
			{MenuItemID: "item-1", Quantity: 2},
			{MenuItemID: "item-2", Quantity: 1},
		},
	}

	same := OrderDraft{
		RestaurantID:    order.RestaurantID,
		DeliveryAddress: order.DeliveryAddress,
		Items: []OrderItem{
			{MenuItemID: "item-2", Quantity: 1},
			{MenuItemID: "item-1", Quantity: 2},
		},
	}

	if !same.Matches(order) {
		t.Error("draft with the same positions must match the order")
	}

	tests := []struct {
		name  string
		draft OrderDraft
	}{
		{
			name: "another restaurant",
			draft: OrderDraft{
				RestaurantID:    "restaurant-2",
				DeliveryAddress: order.DeliveryAddress,
				Items:           order.Items,
			},
		},
		{
			name: "another address",
			draft: OrderDraft{
				RestaurantID:    order.RestaurantID,
				DeliveryAddress: "Батуми, улица Руставели, дом 1",
				Items:           order.Items,
			},
		},
		{
			name: "another quantity",
			draft: OrderDraft{
				RestaurantID:    order.RestaurantID,
				DeliveryAddress: order.DeliveryAddress,
				Items: []OrderItem{
					{MenuItemID: "item-1", Quantity: 3},
					{MenuItemID: "item-2", Quantity: 1},
				},
			},
		},
		{
			name: "another position",
			draft: OrderDraft{
				RestaurantID:    order.RestaurantID,
				DeliveryAddress: order.DeliveryAddress,
				Items: []OrderItem{
					{MenuItemID: "item-1", Quantity: 2},
					{MenuItemID: "item-3", Quantity: 1},
				},
			},
		},
		{
			name: "fewer positions",
			draft: OrderDraft{
				RestaurantID:    order.RestaurantID,
				DeliveryAddress: order.DeliveryAddress,
				Items: []OrderItem{
					{MenuItemID: "item-1", Quantity: 2},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.draft.Matches(order) {
				t.Error("draft must not match the order")
			}
		})
	}
}
