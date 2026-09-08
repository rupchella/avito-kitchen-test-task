package service

import (
	"context"
	"errors"
	"testing"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const (
	testRestaurantID = "a1b2c3d4-0000-4000-8000-000000000001"
	testUserID       = "11111111-1111-4111-8111-111111111111"
	testPizzaID      = "22222222-2222-4222-8222-222222222222"
	testLemonadeID   = "33333333-3333-4333-8333-333333333333"
)

type fakeOrderRepository struct {
	create              func(ctx context.Context, draft domain.OrderDraft) (domain.Order, error)
	getByID             func(ctx context.Context, orderID string) (domain.Order, error)
	getByIdempotencyKey func(ctx context.Context, userID, key string) (domain.Order, error)
	listByUser          func(ctx context.Context, userID string, status *domain.OrderStatus, limit, offset int) ([]domain.Order, int, error)
}

func (f fakeOrderRepository) Create(
	ctx context.Context,
	draft domain.OrderDraft,
) (domain.Order, error) {
	return f.create(ctx, draft)
}

func (f fakeOrderRepository) GetByID(
	ctx context.Context,
	orderID string,
) (domain.Order, error) {
	return f.getByID(ctx, orderID)
}

func (f fakeOrderRepository) GetByIdempotencyKey(
	ctx context.Context,
	userID string,
	idempotencyKey string,
) (domain.Order, error) {
	if f.getByIdempotencyKey == nil {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	return f.getByIdempotencyKey(ctx, userID, idempotencyKey)
}

func (f fakeOrderRepository) ListByUser(
	ctx context.Context,
	userID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	return f.listByUser(ctx, userID, status, limit, offset)
}

type fakeMenuRepository struct {
	items []domain.MenuItem
	err   error
}

func (f fakeMenuRepository) ListForOrder(
	_ context.Context,
	_ string,
	menuItemIDs []string,
) ([]domain.MenuItem, error) {
	if f.err != nil {
		return nil, f.err
	}

	requested := make(map[string]struct{}, len(menuItemIDs))

	for _, id := range menuItemIDs {
		requested[id] = struct{}{}
	}

	found := make([]domain.MenuItem, 0, len(menuItemIDs))

	for _, item := range f.items {
		if _, ok := requested[item.ID]; ok {
			found = append(found, item)
		}
	}

	return found, nil
}

type fakeRestaurantRepository struct {
	err error
}

func (f fakeRestaurantRepository) GetActiveByID(
	_ context.Context,
	id string,
) (domain.Restaurant, error) {
	if f.err != nil {
		return domain.Restaurant{}, f.err
	}

	return domain.Restaurant{
		ID:     id,
		Status: domain.RestaurantStatusActive,
	}, nil
}

func availableMenu() fakeMenuRepository {
	return fakeMenuRepository{
		items: []domain.MenuItem{
			{
				ID:           testPizzaID,
				RestaurantID: testRestaurantID,
				Name:         "Пицца Маргарита",
				PriceKopecks: 49900,
				IsAvailable:  true,
			},
			{
				ID:           testLemonadeID,
				RestaurantID: testRestaurantID,
				Name:         "Домашний лимонад",
				PriceKopecks: 19000,
				IsAvailable:  false,
			},
		},
	}
}

func createInput(items ...CreateOrderItem) CreateOrderInput {
	return CreateOrderInput{
		UserID:          testUserID,
		IdempotencyKey:  "key-1",
		RestaurantID:    testRestaurantID,
		DeliveryAddress: "Батуми, улица Пушкина, дом 10",
		Items:           items,
	}
}

func TestOrderServiceCreateFixesPriceAndTotal(t *testing.T) {
	var saved domain.OrderDraft

	orders := fakeOrderRepository{
		create: func(
			_ context.Context,
			draft domain.OrderDraft,
		) (domain.Order, error) {
			saved = draft

			return domain.Order{
				ID:                "order-1",
				RestaurantID:      draft.RestaurantID,
				UserID:            draft.UserID,
				Status:            domain.OrderStatusCreated,
				TotalPriceKopecks: draft.TotalPriceKopecks(),
				DeliveryAddress:   draft.DeliveryAddress,
				Items:             draft.Items,
			}, nil
		},
	}

	service := NewOrderService(
		orders,
		availableMenu(),
		fakeRestaurantRepository{},
	)

	order, created, err := service.Create(
		t.Context(),
		createInput(CreateOrderItem{MenuItemID: testPizzaID, Quantity: 2}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !created {
		t.Fatal("order must be reported as created")
	}

	if len(saved.Items) != 1 {
		t.Fatalf("expected 1 saved position, got %d", len(saved.Items))
	}

	item := saved.Items[0]

	if item.ItemName != "Пицца Маргарита" {
		t.Errorf("unexpected item name: %q", item.ItemName)
	}

	if item.UnitPriceKopecks != 49900 {
		t.Errorf("unexpected unit price: %d", item.UnitPriceKopecks)
	}

	if order.TotalPriceKopecks != 99800 {
		t.Errorf("unexpected total price: %d", order.TotalPriceKopecks)
	}

	if order.Status != domain.OrderStatusCreated {
		t.Errorf("unexpected status: %q", order.Status)
	}
}

func TestOrderServiceCreateRejectsInvalidItems(t *testing.T) {
	tests := []struct {
		name     string
		items    []CreateOrderItem
		expected error
	}{
		{
			name: "unavailable item",
			items: []CreateOrderItem{
				{MenuItemID: testLemonadeID, Quantity: 1},
			},
			expected: domain.ErrMenuItemUnavailable,
		},
		{
			name: "unknown item",
			items: []CreateOrderItem{
				{MenuItemID: "44444444-4444-4444-8444-444444444444", Quantity: 1},
			},
			expected: domain.ErrMenuItemNotFound,
		},
		{
			name: "duplicated item",
			items: []CreateOrderItem{
				{MenuItemID: testPizzaID, Quantity: 1},
				{MenuItemID: testPizzaID, Quantity: 2},
			},
			expected: domain.ErrDuplicateMenuItem,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			orders := fakeOrderRepository{
				create: func(
					_ context.Context,
					_ domain.OrderDraft,
				) (domain.Order, error) {
					t.Fatal("order must not be created")

					return domain.Order{}, nil
				},
			}

			service := NewOrderService(
				orders,
				availableMenu(),
				fakeRestaurantRepository{},
			)

			_, _, err := service.Create(t.Context(), createInput(test.items...))

			if !errors.Is(err, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, err)
			}
		})
	}
}

func TestOrderServiceCreateRejectsInactiveRestaurant(t *testing.T) {
	service := NewOrderService(
		fakeOrderRepository{},
		availableMenu(),
		fakeRestaurantRepository{err: domain.ErrRestaurantNotFound},
	)

	_, _, err := service.Create(
		t.Context(),
		createInput(CreateOrderItem{MenuItemID: testPizzaID, Quantity: 1}),
	)

	if !errors.Is(err, domain.ErrRestaurantNotFound) {
		t.Fatalf("expected restaurant not found, got %v", err)
	}
}

func TestOrderServiceCreateReturnsExistingOrderForRepeatedRequest(t *testing.T) {
	existing := domain.Order{
		ID:              "order-1",
		RestaurantID:    testRestaurantID,
		UserID:          testUserID,
		Status:          domain.OrderStatusAccepted,
		DeliveryAddress: "Батуми, улица Пушкина, дом 10",
		Items: []domain.OrderItem{
			{MenuItemID: testPizzaID, Quantity: 2},
		},
	}

	orders := fakeOrderRepository{
		getByIdempotencyKey: func(
			_ context.Context,
			_ string,
			_ string,
		) (domain.Order, error) {
			return existing, nil
		},
		create: func(
			_ context.Context,
			_ domain.OrderDraft,
		) (domain.Order, error) {
			t.Fatal("repeated request must not create a new order")

			return domain.Order{}, nil
		},
	}

	service := NewOrderService(
		orders,
		availableMenu(),
		fakeRestaurantRepository{},
	)

	order, created, err := service.Create(
		t.Context(),
		createInput(CreateOrderItem{MenuItemID: testPizzaID, Quantity: 2}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created {
		t.Error("repeated request must not report a new order")
	}

	if order.ID != existing.ID {
		t.Errorf("unexpected order: %q", order.ID)
	}
}

func TestOrderServiceCreateRejectsReusedIdempotencyKey(t *testing.T) {
	orders := fakeOrderRepository{
		getByIdempotencyKey: func(
			_ context.Context,
			_ string,
			_ string,
		) (domain.Order, error) {
			return domain.Order{
				ID:              "order-1",
				RestaurantID:    testRestaurantID,
				UserID:          testUserID,
				DeliveryAddress: "Батуми, улица Пушкина, дом 10",
				Items: []domain.OrderItem{
					{MenuItemID: testPizzaID, Quantity: 1},
				},
			}, nil
		},
	}

	service := NewOrderService(
		orders,
		availableMenu(),
		fakeRestaurantRepository{},
	)

	_, _, err := service.Create(
		t.Context(),
		createInput(CreateOrderItem{MenuItemID: testPizzaID, Quantity: 5}),
	)

	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestOrderServiceCreateResolvesConcurrentIdempotentRequest(t *testing.T) {
	existing := domain.Order{
		ID:              "order-1",
		RestaurantID:    testRestaurantID,
		UserID:          testUserID,
		DeliveryAddress: "Батуми, улица Пушкина, дом 10",
		Items: []domain.OrderItem{
			{MenuItemID: testPizzaID, Quantity: 1},
		},
	}

	lookups := 0

	orders := fakeOrderRepository{
		getByIdempotencyKey: func(
			_ context.Context,
			_ string,
			_ string,
		) (domain.Order, error) {
			lookups++

			if lookups == 1 {
				return domain.Order{}, domain.ErrOrderNotFound
			}

			return existing, nil
		},
		create: func(
			_ context.Context,
			_ domain.OrderDraft,
		) (domain.Order, error) {
			return domain.Order{}, domain.ErrIdempotencyConflict
		},
	}

	service := NewOrderService(
		orders,
		availableMenu(),
		fakeRestaurantRepository{},
	)

	order, created, err := service.Create(
		t.Context(),
		createInput(CreateOrderItem{MenuItemID: testPizzaID, Quantity: 1}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created {
		t.Error("order must not be reported as created")
	}

	if order.ID != existing.ID {
		t.Errorf("unexpected order: %q", order.ID)
	}
}

func TestOrderServiceGetHidesForeignOrder(t *testing.T) {
	orders := fakeOrderRepository{
		getByID: func(
			_ context.Context,
			_ string,
		) (domain.Order, error) {
			return domain.Order{
				ID:     "order-1",
				UserID: "99999999-9999-4999-8999-999999999999",
			}, nil
		},
	}

	service := NewOrderService(
		orders,
		availableMenu(),
		fakeRestaurantRepository{},
	)

	_, err := service.Get(t.Context(), "order-1", testUserID)

	if !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("expected order not found, got %v", err)
	}
}
