package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/domain"
)

const (
	uniqueViolationCode        = "23505"
	orderIdempotencyConstraint = "orders_user_idempotency_key"
)

const orderColumns = `
	id::text,
	restaurant_id::text,
	user_id,
	status,
	total_price_kopecks,
	delivery_address,
	created_at,
	updated_at
`

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{
		pool: pool,
	}
}

func (r *OrderRepository) Create(
	ctx context.Context,
	draft domain.OrderDraft,
) (domain.Order, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin order creation: %w", err)
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	insertOrderQuery := `
		INSERT INTO orders (
			restaurant_id,
			user_id,
			idempotency_key,
			status,
			total_price_kopecks,
			delivery_address
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING` + orderColumns

	var order domain.Order
	var status string

	err = transaction.QueryRow(
		ctx,
		insertOrderQuery,
		draft.RestaurantID,
		draft.UserID,
		draft.IdempotencyKey,
		string(domain.OrderStatusCreated),
		draft.TotalPriceKopecks(),
		draft.DeliveryAddress,
	).Scan(
		&order.ID,
		&order.RestaurantID,
		&order.UserID,
		&status,
		&order.TotalPriceKopecks,
		&order.DeliveryAddress,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if isIdempotencyViolation(err) {
		return domain.Order{}, domain.ErrIdempotencyConflict
	}

	if err != nil {
		return domain.Order{}, fmt.Errorf("insert order: %w", err)
	}

	order.Status = domain.OrderStatus(status)

	menuItemIDs := make([]string, 0, len(draft.Items))
	itemNames := make([]string, 0, len(draft.Items))
	unitPrices := make([]int64, 0, len(draft.Items))
	quantities := make([]int64, 0, len(draft.Items))

	for _, item := range draft.Items {
		menuItemIDs = append(menuItemIDs, item.MenuItemID)
		itemNames = append(itemNames, item.ItemName)
		unitPrices = append(unitPrices, item.UnitPriceKopecks)
		quantities = append(quantities, int64(item.Quantity))
	}

	const insertItemsQuery = `
		INSERT INTO order_items (
			order_id,
			menu_item_id,
			item_name,
			unit_price_kopecks,
			quantity
		)
		SELECT
			$1,
			input.menu_item_id,
			input.item_name,
			input.unit_price_kopecks,
			input.quantity
		FROM unnest(
			$2::uuid[],
			$3::text[],
			$4::bigint[],
			$5::bigint[]
		) AS input (
			menu_item_id,
			item_name,
			unit_price_kopecks,
			quantity
		)
	`

	if _, err := transaction.Exec(
		ctx,
		insertItemsQuery,
		order.ID,
		menuItemIDs,
		itemNames,
		unitPrices,
		quantities,
	); err != nil {
		return domain.Order{}, fmt.Errorf("insert order items: %w", err)
	}

	items, err := loadOrderItems(ctx, transaction, []string{order.ID})
	if err != nil {
		return domain.Order{}, err
	}

	order.Items = items[order.ID]

	if err := transaction.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit order creation: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) GetByID(
	ctx context.Context,
	orderID string,
) (domain.Order, error) {
	query := `
		SELECT` + orderColumns + `
		FROM orders
		WHERE id = $1
	`

	return r.getOne(ctx, query, orderID)
}

func (r *OrderRepository) GetByIdempotencyKey(
	ctx context.Context,
	userID string,
	idempotencyKey string,
) (domain.Order, error) {
	query := `
		SELECT` + orderColumns + `
		FROM orders
		WHERE user_id = $1
			AND idempotency_key = $2
	`

	return r.getOne(ctx, query, userID, idempotencyKey)
}

func (r *OrderRepository) ListByUser(
	ctx context.Context,
	userID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	const filter = `
		WHERE user_id = $1
			AND ($2::text IS NULL OR status = $2)
	`

	return r.list(ctx, filter, userID, statusFilterValue(status), limit, offset)
}

func (r *OrderRepository) ListByRestaurant(
	ctx context.Context,
	restaurantID string,
	status *domain.OrderStatus,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	const filter = `
		WHERE restaurant_id = $1
			AND ($2::text IS NULL OR status = $2)
	`

	return r.list(ctx, filter, restaurantID, statusFilterValue(status), limit, offset)
}

// UpdateStatus переводит заказ заведения в новый статус.
func (r *OrderRepository) UpdateStatus(
	ctx context.Context,
	orderID string,
	restaurantID string,
	target domain.OrderStatus,
) (domain.Order, error) {
	const lockQuery = `
		SELECT status
		FROM orders
		WHERE id = $1
			AND restaurant_id = $2
		FOR UPDATE
	`

	return r.changeStatus(
		ctx,
		lockQuery,
		[]any{orderID, restaurantID},
		orderID,
		target,
	)
}

// CancelByUser отменяет заказ по требованию пользователя.
// Заказ ищется вместе с владельцем, поэтому чужой заказ отменить нельзя.
func (r *OrderRepository) CancelByUser(
	ctx context.Context,
	orderID string,
	userID string,
) (domain.Order, error) {
	const lockQuery = `
		SELECT status
		FROM orders
		WHERE id = $1
			AND user_id = $2
		FOR UPDATE
	`

	return r.changeStatus(
		ctx,
		lockQuery,
		[]any{orderID, userID},
		orderID,
		domain.OrderStatusCancelled,
	)
}

// changeStatus выполняет переход статуса под блокировкой строки:
// текущий статус читается через FOR UPDATE, поэтому два параллельных
// запроса не могут выполнить два перехода одновременно.
func (r *OrderRepository) changeStatus(
	ctx context.Context,
	lockQuery string,
	lockArguments []any,
	orderID string,
	target domain.OrderStatus,
) (domain.Order, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin status update: %w", err)
	}

	defer func() {
		_ = transaction.Rollback(ctx)
	}()

	var currentStatus string

	err = transaction.QueryRow(ctx, lockQuery, lockArguments...).Scan(&currentStatus)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	if err != nil {
		return domain.Order{}, fmt.Errorf("lock order: %w", err)
	}

	if !domain.OrderStatus(currentStatus).CanTransitionTo(target) {
		return domain.Order{}, domain.ErrOrderStatusForbidden
	}

	updateQuery := `
		UPDATE orders
		SET
			status = $2,
			updated_at = NOW()
		WHERE id = $1
		RETURNING` + orderColumns

	var order domain.Order
	var status string

	if err := transaction.QueryRow(
		ctx,
		updateQuery,
		orderID,
		string(target),
	).Scan(
		&order.ID,
		&order.RestaurantID,
		&order.UserID,
		&status,
		&order.TotalPriceKopecks,
		&order.DeliveryAddress,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return domain.Order{}, fmt.Errorf("update order status: %w", err)
	}

	order.Status = domain.OrderStatus(status)

	items, err := loadOrderItems(ctx, transaction, []string{order.ID})
	if err != nil {
		return domain.Order{}, err
	}

	order.Items = items[order.ID]

	if err := transaction.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit status update: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) getOne(
	ctx context.Context,
	query string,
	arguments ...any,
) (domain.Order, error) {
	var order domain.Order
	var status string

	err := r.pool.QueryRow(ctx, query, arguments...).Scan(
		&order.ID,
		&order.RestaurantID,
		&order.UserID,
		&status,
		&order.TotalPriceKopecks,
		&order.DeliveryAddress,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	if err != nil {
		return domain.Order{}, fmt.Errorf("get order: %w", err)
	}

	order.Status = domain.OrderStatus(status)

	items, err := loadOrderItems(ctx, r.pool, []string{order.ID})
	if err != nil {
		return domain.Order{}, err
	}

	order.Items = items[order.ID]

	return order, nil
}

func (r *OrderRepository) list(
	ctx context.Context,
	filter string,
	owner string,
	status *string,
	limit int,
	offset int,
) ([]domain.Order, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM orders
	` + filter

	var total int

	if err := r.pool.QueryRow(ctx, countQuery, owner, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	listQuery := `
		SELECT` + orderColumns + `
		FROM orders
	` + filter + `
		ORDER BY created_at DESC, id
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.pool.Query(ctx, listQuery, owner, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query orders: %w", err)
	}

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, 0, err
	}

	orderIDs := make([]string, 0, len(orders))

	for _, order := range orders {
		orderIDs = append(orderIDs, order.ID)
	}

	items, err := loadOrderItems(ctx, r.pool, orderIDs)
	if err != nil {
		return nil, 0, err
	}

	for index := range orders {
		orders[index].Items = items[orders[index].ID]
	}

	return orders, total, nil
}

type querier interface {
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

func loadOrderItems(
	ctx context.Context,
	db querier,
	orderIDs []string,
) (map[string][]domain.OrderItem, error) {
	items := make(map[string][]domain.OrderItem, len(orderIDs))

	if len(orderIDs) == 0 {
		return items, nil
	}

	const query = `
		SELECT
			order_id::text,
			menu_item_id::text,
			item_name,
			unit_price_kopecks,
			quantity
		FROM order_items
		WHERE order_id = ANY($1::uuid[])
		ORDER BY item_name, menu_item_id
	`

	rows, err := db.Query(ctx, query, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item domain.OrderItem

		if err := rows.Scan(
			&orderID,
			&item.MenuItemID,
			&item.ItemName,
			&item.UnitPriceKopecks,
			&item.Quantity,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}

		items[orderID] = append(items[orderID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return items, nil
}

func scanOrders(rows pgx.Rows) ([]domain.Order, error) {
	defer rows.Close()

	orders := make([]domain.Order, 0)

	for rows.Next() {
		var order domain.Order
		var status string

		if err := rows.Scan(
			&order.ID,
			&order.RestaurantID,
			&order.UserID,
			&status,
			&order.TotalPriceKopecks,
			&order.DeliveryAddress,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		order.Status = domain.OrderStatus(status)
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func statusFilterValue(status *domain.OrderStatus) *string {
	if status == nil {
		return nil
	}

	value := string(*status)

	return &value
}

func isIdempotencyViolation(err error) bool {
	var pgError *pgconn.PgError

	if !errors.As(err, &pgError) {
		return false
	}

	return pgError.Code == uniqueViolationCode &&
		pgError.ConstraintName == orderIdempotencyConstraint
}
