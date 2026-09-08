package httpserver

import (
	"context"
	"io"
	"log/slog"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/service"
)

type fakeRestaurantService struct {
	listActive func(
		ctx context.Context,
		limit int,
		offset int,
	) ([]domain.Restaurant, int, error)

	get func(
		ctx context.Context,
		id string,
	) (domain.Restaurant, error)

	menu func(
		ctx context.Context,
		restaurantID string,
		search string,
		limit int,
		offset int,
	) (domain.Restaurant, []domain.MenuItem, int, error)

	searchMenuItems func(
		ctx context.Context,
		search string,
		limit int,
		offset int,
	) ([]domain.MenuSearchItem, int, error)
}

func (f fakeRestaurantService) ListActive(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.Restaurant, int, error) {
	return f.listActive(ctx, limit, offset)
}

func (f fakeRestaurantService) Get(
	ctx context.Context,
	id string,
) (domain.Restaurant, error) {
	return f.get(ctx, id)
}

func (f fakeRestaurantService) Menu(
	ctx context.Context,
	restaurantID string,
	search string,
	limit int,
	offset int,
) (domain.Restaurant, []domain.MenuItem, int, error) {
	return f.menu(ctx, restaurantID, search, limit, offset)
}

func (f fakeRestaurantService) SearchMenuItems(
	ctx context.Context,
	search string,
	limit int,
	offset int,
) ([]domain.MenuSearchItem, int, error) {
	return f.searchMenuItems(ctx, search, limit, offset)
}

type fakeOrderService struct {
	create func(
		ctx context.Context,
		input service.CreateOrderInput,
	) (domain.Order, bool, error)

	get func(
		ctx context.Context,
		orderID string,
		userID string,
	) (domain.Order, error)

	list func(
		ctx context.Context,
		userID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)

	cancel func(
		ctx context.Context,
		orderID string,
		userID string,
	) (domain.Order, error)
}

func (f fakeOrderService) Create(
	ctx context.Context,
	input service.CreateOrderInput,
) (domain.Order, bool, error) {
	return f.create(ctx, input)
}

func (f fakeOrderService) Get(
	ctx context.Context,
	orderID string,
	userID string,
) (domain.Order, error) {
	return f.get(ctx, orderID, userID)
}

func (f fakeOrderService) Cancel(
	ctx context.Context,
	orderID string,
	userID string,
) (domain.Order, error) {
	return f.cancel(ctx, orderID, userID)
}

func (f fakeOrderService) List(
	ctx context.Context,
	userID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	return f.list(ctx, userID, status, limit, offset)
}

type fakePartnerService struct {
	authenticate func(
		ctx context.Context,
		apiKey string,
	) (string, error)

	syncMenu func(
		ctx context.Context,
		restaurantID string,
		inputs []domain.MenuItemInput,
	) ([]domain.MenuItem, error)

	setAvailability func(
		ctx context.Context,
		restaurantID string,
		externalID string,
		isAvailable bool,
	) (domain.MenuItem, error)

	listOrders func(
		ctx context.Context,
		restaurantID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)

	updateOrderStatus func(
		ctx context.Context,
		restaurantID string,
		orderID string,
		target domain.OrderStatus,
	) (domain.Order, error)
}

func (f fakePartnerService) Authenticate(
	ctx context.Context,
	apiKey string,
) (string, error) {
	return f.authenticate(ctx, apiKey)
}

func (f fakePartnerService) SyncMenu(
	ctx context.Context,
	restaurantID string,
	inputs []domain.MenuItemInput,
) ([]domain.MenuItem, error) {
	return f.syncMenu(ctx, restaurantID, inputs)
}

func (f fakePartnerService) SetMenuItemAvailability(
	ctx context.Context,
	restaurantID string,
	externalID string,
	isAvailable bool,
) (domain.MenuItem, error) {
	return f.setAvailability(ctx, restaurantID, externalID, isAvailable)
}

func (f fakePartnerService) ListOrders(
	ctx context.Context,
	restaurantID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	return f.listOrders(ctx, restaurantID, status, limit, offset)
}

func (f fakePartnerService) UpdateOrderStatus(
	ctx context.Context,
	restaurantID string,
	orderID string,
	target domain.OrderStatus,
) (domain.Order, error) {
	return f.updateOrderStatus(ctx, restaurantID, orderID, target)
}

func testLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(io.Discard, nil),
	)
}
