package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const menuItemColumns = `
	id::text,
	restaurant_id::text,
	external_id,
	name,
	description,
	price_kopecks,
	is_available,
	created_at,
	updated_at
`

type MenuRepository struct {
	pool *pgxpool.Pool
}

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{
		pool: pool,
	}
}

func (r *MenuRepository) ListByRestaurant(
	ctx context.Context,
	restaurantID string,
	limit int,
	offset int,
) ([]domain.MenuItem, int, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM menu_items
		WHERE restaurant_id = $1
	`

	var total int

	if err := r.pool.QueryRow(ctx, countQuery, restaurantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count menu items: %w", err)
	}

	listQuery := `
		SELECT` + menuItemColumns + `
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY name, id
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.pool.Query(ctx, listQuery, restaurantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query menu items: %w", err)
	}

	items, err := scanMenuItems(rows)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *MenuRepository) ListForOrder(
	ctx context.Context,
	restaurantID string,
	menuItemIDs []string,
) ([]domain.MenuItem, error) {
	query := `
		SELECT` + menuItemColumns + `
		FROM menu_items
		WHERE restaurant_id = $1
			AND id = ANY($2::uuid[])
	`

	rows, err := r.pool.Query(ctx, query, restaurantID, menuItemIDs)
	if err != nil {
		return nil, fmt.Errorf("query menu items for order: %w", err)
	}

	items, err := scanMenuItems(rows)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *MenuRepository) Sync(
	ctx context.Context,
	restaurantID string,
	inputs []domain.MenuItemInput,
) ([]domain.MenuItem, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin menu sync: %w", err)
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	externalIDs := make([]string, 0, len(inputs))
	names := make([]string, 0, len(inputs))
	descriptions := make([]string, 0, len(inputs))
	prices := make([]int64, 0, len(inputs))
	availability := make([]bool, 0, len(inputs))

	for _, input := range inputs {
		externalIDs = append(externalIDs, input.ExternalID)
		names = append(names, input.Name)
		descriptions = append(descriptions, input.Description)
		prices = append(prices, input.PriceKopecks)
		availability = append(availability, input.IsAvailable)
	}

	const upsertQuery = `
		INSERT INTO menu_items (
			restaurant_id,
			external_id,
			name,
			description,
			price_kopecks,
			is_available
		)
		SELECT
			$1,
			input.external_id,
			input.name,
			input.description,
			input.price_kopecks,
			input.is_available
		FROM unnest(
			$2::text[],
			$3::text[],
			$4::text[],
			$5::bigint[],
			$6::boolean[]
		) AS input (
			external_id,
			name,
			description,
			price_kopecks,
			is_available
		)
		ON CONFLICT (restaurant_id, external_id) DO UPDATE
		SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			price_kopecks = EXCLUDED.price_kopecks,
			is_available = EXCLUDED.is_available,
			updated_at = NOW()
	`

	if _, err := transaction.Exec(
		ctx,
		upsertQuery,
		restaurantID,
		externalIDs,
		names,
		descriptions,
		prices,
		availability,
	); err != nil {
		return nil, fmt.Errorf("upsert menu items: %w", err)
	}

	const deactivateQuery = `
		UPDATE menu_items
		SET
			is_available = FALSE,
			updated_at = NOW()
		WHERE restaurant_id = $1
			AND NOT (external_id = ANY($2::text[]))
			AND is_available
	`

	if _, err := transaction.Exec(
		ctx,
		deactivateQuery,
		restaurantID,
		externalIDs,
	); err != nil {
		return nil, fmt.Errorf("deactivate missing menu items: %w", err)
	}

	listQuery := `
		SELECT` + menuItemColumns + `
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY name, id
	`

	rows, err := transaction.Query(ctx, listQuery, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("query synchronized menu: %w", err)
	}

	items, err := scanMenuItems(rows)
	if err != nil {
		return nil, err
	}

	if err := transaction.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit menu sync: %w", err)
	}

	return items, nil
}

func (r *MenuRepository) SetAvailability(
	ctx context.Context,
	restaurantID string,
	externalID string,
	isAvailable bool,
) (domain.MenuItem, error) {
	query := `
		UPDATE menu_items
		SET
			is_available = $3,
			updated_at = NOW()
		WHERE restaurant_id = $1
			AND external_id = $2
		RETURNING` + menuItemColumns

	rows, err := r.pool.Query(
		ctx,
		query,
		restaurantID,
		externalID,
		isAvailable,
	)
	if err != nil {
		return domain.MenuItem{}, fmt.Errorf("update menu item availability: %w", err)
	}

	items, err := scanMenuItems(rows)
	if err != nil {
		return domain.MenuItem{}, err
	}

	if len(items) == 0 {
		return domain.MenuItem{}, domain.ErrMenuItemNotFound
	}

	return items[0], nil
}

func scanMenuItems(rows pgx.Rows) ([]domain.MenuItem, error) {
	defer rows.Close()

	items := make([]domain.MenuItem, 0)

	for rows.Next() {
		var item domain.MenuItem

		if err := rows.Scan(
			&item.ID,
			&item.RestaurantID,
			&item.ExternalID,
			&item.Name,
			&item.Description,
			&item.PriceKopecks,
			&item.IsAvailable,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan menu item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate menu items: %w", err)
	}

	return items, nil
}
