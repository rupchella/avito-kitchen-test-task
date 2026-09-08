package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

type IntegrationRepository struct {
	pool *pgxpool.Pool
}

func NewIntegrationRepository(pool *pgxpool.Pool) *IntegrationRepository {
	return &IntegrationRepository{
		pool: pool,
	}
}

func (r *IntegrationRepository) RestaurantIDByAPIKeyHash(
	ctx context.Context,
	apiKeyHash string,
) (string, error) {
	const query = `
		SELECT restaurant_id::text
		FROM restaurant_integrations
		WHERE api_key_hash = $1
	`

	var restaurantID string

	err := r.pool.QueryRow(ctx, query, apiKeyHash).Scan(&restaurantID)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrPartnerUnauthorized
	}

	if err != nil {
		return "", fmt.Errorf("get restaurant by api key: %w", err)
	}

	return restaurantID, nil
}
