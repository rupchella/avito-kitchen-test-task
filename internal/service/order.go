package service

import (
	"context"
	"errors"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type OrderRepository interface {
	Create(
		ctx context.Context,
		draft domain.OrderDraft,
	) (domain.Order, error)

	GetByID(
		ctx context.Context,
		orderID string,
	) (domain.Order, error)

	GetByIdempotencyKey(
		ctx context.Context,
		userID string,
		idempotencyKey string,
	) (domain.Order, error)

	ListByUser(
		ctx context.Context,
		userID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)
}

type OrderMenuRepository interface {
	ListForOrder(
		ctx context.Context,
		restaurantID string,
		menuItemIDs []string,
	) ([]domain.MenuItem, error)
}

type OrderRestaurantRepository interface {
	GetActiveByID(
		ctx context.Context,
		id string,
	) (domain.Restaurant, error)
}

type CreateOrderItem struct {
	MenuItemID string
	Quantity   int
}

type CreateOrderInput struct {
	UserID          string
	IdempotencyKey  string
	RestaurantID    string
	DeliveryAddress string
	Items           []CreateOrderItem
}

type OrderService struct {
	orders      OrderRepository
	menu        OrderMenuRepository
	restaurants OrderRestaurantRepository
}

func NewOrderService(
	orders OrderRepository,
	menu OrderMenuRepository,
	restaurants OrderRestaurantRepository,
) *OrderService {
	return &OrderService{
		orders:      orders,
		menu:        menu,
		restaurants: restaurants,
	}
}

func (s *OrderService) Create(
	ctx context.Context,
	input CreateOrderInput,
) (domain.Order, bool, error) {
	draft := domain.OrderDraft{
		RestaurantID:    input.RestaurantID,
		UserID:          input.UserID,
		IdempotencyKey:  input.IdempotencyKey,
		DeliveryAddress: input.DeliveryAddress,
		Items:           make([]domain.OrderItem, 0, len(input.Items)),
	}

	for _, item := range input.Items {
		draft.Items = append(draft.Items, domain.OrderItem{
			MenuItemID: item.MenuItemID,
			Quantity:   item.Quantity,
		})
	}

	existing, err := s.existingOrder(ctx, draft)
	if err != nil {
		return domain.Order{}, false, err
	}

	if existing != nil {
		return *existing, false, nil
	}

	if _, err := s.restaurants.GetActiveByID(ctx, input.RestaurantID); err != nil {
		return domain.Order{}, false, err
	}

	if err := s.fillOrderItems(ctx, &draft); err != nil {
		return domain.Order{}, false, err
	}

	order, err := s.orders.Create(ctx, draft)

	if errors.Is(err, domain.ErrIdempotencyConflict) {
		existing, existingErr := s.existingOrder(ctx, draft)
		if existingErr != nil {
			return domain.Order{}, false, existingErr
		}

		if existing != nil {
			return *existing, false, nil
		}

		return domain.Order{}, false, domain.ErrIdempotencyConflict
	}

	if err != nil {
		return domain.Order{}, false, err
	}

	return order, true, nil
}

func (s *OrderService) Get(
	ctx context.Context,
	orderID string,
	userID string,
) (domain.Order, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}

	if order.UserID != userID {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	return order, nil
}

func (s *OrderService) List(
	ctx context.Context,
	userID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	return s.orders.ListByUser(ctx, userID, status, limit, offset)
}

func (s *OrderService) existingOrder(
	ctx context.Context,
	draft domain.OrderDraft,
) (*domain.Order, error) {
	order, err := s.orders.GetByIdempotencyKey(
		ctx,
		draft.UserID,
		draft.IdempotencyKey,
	)

	if errors.Is(err, domain.ErrOrderNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if !draft.Matches(order) {
		return nil, domain.ErrIdempotencyConflict
	}

	return &order, nil
}

func (s *OrderService) fillOrderItems(
	ctx context.Context,
	draft *domain.OrderDraft,
) error {
	menuItemIDs := make([]string, 0, len(draft.Items))
	seen := make(map[string]struct{}, len(draft.Items))

	for _, item := range draft.Items {
		if _, duplicate := seen[item.MenuItemID]; duplicate {
			return domain.ErrDuplicateMenuItem
		}

		seen[item.MenuItemID] = struct{}{}
		menuItemIDs = append(menuItemIDs, item.MenuItemID)
	}

	menuItems, err := s.menu.ListForOrder(ctx, draft.RestaurantID, menuItemIDs)
	if err != nil {
		return err
	}

	byID := make(map[string]domain.MenuItem, len(menuItems))

	for _, menuItem := range menuItems {
		byID[menuItem.ID] = menuItem
	}

	for index, item := range draft.Items {
		menuItem, found := byID[item.MenuItemID]
		if !found {
			return domain.ErrMenuItemNotFound
		}

		if !menuItem.IsAvailable {
			return domain.ErrMenuItemUnavailable
		}

		draft.Items[index].ItemName = menuItem.Name
		draft.Items[index].UnitPriceKopecks = menuItem.PriceKopecks
	}

	return nil
}
