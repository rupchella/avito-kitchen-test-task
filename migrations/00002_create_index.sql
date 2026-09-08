-- +goose Up

CREATE INDEX idx_menu_items_restaurant_available
    ON menu_items (restaurant_id, is_available);

CREATE INDEX idx_orders_restaurant_status_created_at
    ON orders (restaurant_id, status, created_at);

CREATE INDEX idx_orders_user_created_at
    ON orders (user_id, created_at DESC);

CREATE INDEX idx_order_items_menu_item_id
    ON order_items (menu_item_id);

-- +goose Down

DROP INDEX idx_order_items_menu_item_id;
DROP INDEX idx_orders_user_created_at;
DROP INDEX idx_orders_restaurant_status_created_at;
DROP INDEX idx_menu_items_restaurant_available;