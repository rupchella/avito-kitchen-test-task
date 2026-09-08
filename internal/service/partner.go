package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type IntegrationRepository interface {
	RestaurantIDByAPIKeyHash(
		ctx context.Context,
		apiKeyHash string,
	) (string, error)
}

type PartnerMenuRepository interface {
	Sync(
		ctx context.Context,
		restaurantID string,
		inputs []domain.MenuItemInput,
	) ([]domain.MenuItem, error)

	SetAvailability(
		ctx context.Context,
		restaurantID string,
		externalID string,
		isAvailable bool,
	) (domain.MenuItem, error)
}

type PartnerOrderRepository interface {
	ListByRestaurant(
		ctx context.Context,
		restaurantID string,
		status *domain.OrderStatus,
		limit int,
		offset int,
	) ([]domain.Order, int, error)

	UpdateStatus(
		ctx context.Context,
		orderID string,
		restaurantID string,
		target domain.OrderStatus,
	) (domain.Order, error)
}

type PartnerService struct {
	integrations IntegrationRepository
	menu         PartnerMenuRepository
	orders       PartnerOrderRepository
}

func NewPartnerService(
	integrations IntegrationRepository,
	menu PartnerMenuRepository,
	orders PartnerOrderRepository,
) *PartnerService {
	return &PartnerService{
		integrations: integrations,
		menu:         menu,
		orders:       orders,
	}
}

func (s *PartnerService) Authenticate(
	ctx context.Context,
	apiKey string,
) (string, error) {
	if apiKey == "" {
		return "", domain.ErrPartnerUnauthorized
	}

	return s.integrations.RestaurantIDByAPIKeyHash(ctx, HashAPIKey(apiKey))
}

func (s *PartnerService) SyncMenu(
	ctx context.Context,
	restaurantID string,
	inputs []domain.MenuItemInput,
) ([]domain.MenuItem, error) {
	seen := make(map[string]struct{}, len(inputs))

	for _, input := range inputs {
		if _, duplicate := seen[input.ExternalID]; duplicate {
			return nil, domain.ErrDuplicateMenuItem
		}

		seen[input.ExternalID] = struct{}{}
	}

	return s.menu.Sync(ctx, restaurantID, inputs)
}

func (s *PartnerService) SetMenuItemAvailability(
	ctx context.Context,
	restaurantID string,
	externalID string,
	isAvailable bool,
) (domain.MenuItem, error) {
	return s.menu.SetAvailability(ctx, restaurantID, externalID, isAvailable)
}

func (s *PartnerService) ListOrders(
	ctx context.Context,
	restaurantID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	return s.orders.ListByRestaurant(ctx, restaurantID, status, limit, offset)
}

func (s *PartnerService) UpdateOrderStatus(
	ctx context.Context,
	restaurantID string,
	orderID string,
	target domain.OrderStatus,
) (domain.Order, error) {
	return s.orders.UpdateStatus(ctx, orderID, restaurantID, target)
}

func HashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))

	return hex.EncodeToString(sum[:])
}
