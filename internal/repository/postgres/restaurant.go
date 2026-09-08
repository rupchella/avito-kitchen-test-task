package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type RestaurantRepository struct {
	pool *pgxpool.Pool
}

func NewRestaurantRepository(pool *pgxpool.Pool) *RestaurantRepository {
	return &RestaurantRepository{
		pool: pool,
	}
}

func (r *RestaurantRepository) ListActive(
	ctx context.Context,
	limit int,
	offset int,
) ([]domain.Restaurant, int, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM restaurants
		WHERE status = 'active'
	`

	var total int

	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count active restaurants: %w", err)
	}

	const listQuery = `
		SELECT
			id::text,
			name,
			status,
			created_at,
			updated_at
		FROM restaurants
		WHERE status = 'active'
		ORDER BY name, id
		LIMIT $1
		OFFSET $2
	`

	rows, err := r.pool.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query active restaurants: %w", err)
	}
	defer rows.Close()

	restaurants := make([]domain.Restaurant, 0)

	for rows.Next() {
		var restaurant domain.Restaurant
		var status string

		if err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&status,
			&restaurant.CreatedAt,
			&restaurant.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan restaurant: %w", err)
		}

		restaurant.Status = domain.RestaurantStatus(status)
		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate restaurants: %w", err)
	}

	return restaurants, total, nil
}

func (r *RestaurantRepository) GetActiveByID(
	ctx context.Context,
	id string,
) (domain.Restaurant, error) {
	const query = `
		SELECT
			id::text,
			name,
			status,
			created_at,
			updated_at
		FROM restaurants
		WHERE id = $1
			AND status = 'active'
	`

	var restaurant domain.Restaurant
	var status string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&restaurant.ID,
		&restaurant.Name,
		&status,
		&restaurant.CreatedAt,
		&restaurant.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Restaurant{}, domain.ErrRestaurantNotFound
	}

	if err != nil {
		return domain.Restaurant{}, fmt.Errorf("get restaurant: %w", err)
	}

	restaurant.Status = domain.RestaurantStatus(status)

	return restaurant, nil
}
