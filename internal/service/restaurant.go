package service

import (
	"context"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type RestaurantRepository interface {
	ListActive(
		ctx context.Context,
		limit int,
		offset int,
	) ([]domain.Restaurant, int, error)

	GetActiveByID(
		ctx context.Context,
		id string,
	) (domain.Restaurant, error)
}

type MenuRepository interface {
	ListByRestaurant(
		ctx context.Context,
		restaurantID string,
		limit int,
		offset int,
	) ([]domain.MenuItem, int, error)
}

type RestaurantService struct {
	repository     RestaurantRepository
	menuRepository MenuRepository
}

func NewRestaurantService(
	repository RestaurantRepository,
	menuRepository MenuRepository,
) *RestaurantService {
	return &RestaurantService{
		repository:     repository,
		menuRepository: menuRepository,
	}
}

func (s *RestaurantService) ListActive(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.Restaurant, int, error) {
	return s.repository.ListActive(ctx, limit, offset)
}

func (s *RestaurantService) Get(
	ctx context.Context,
	id string,
) (domain.Restaurant, error) {
	return s.repository.GetActiveByID(ctx, id)
}

func (s *RestaurantService) Menu(
	ctx context.Context,
	restaurantID string,
	limit int,
	offset int,
) (domain.Restaurant, []domain.MenuItem, int, error) {
	restaurant, err := s.repository.GetActiveByID(ctx, restaurantID)
	if err != nil {
		return domain.Restaurant{}, nil, 0, err
	}

	items, total, err := s.menuRepository.ListByRestaurant(
		ctx,
		restaurantID,
		limit,
		offset,
	)
	if err != nil {
		return domain.Restaurant{}, nil, 0, err
	}

	return restaurant, items, total, nil
}
