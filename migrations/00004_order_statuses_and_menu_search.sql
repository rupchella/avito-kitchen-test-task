-- +goose Up

ALTER TABLE orders
    DROP CONSTRAINT orders_status_check;

ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
        CHECK (
            status IN (
                       'created',
                       'accepted',
                       'preparing',
                       'ready',
                       'delivered',
                       'rejected',
                       'cancelled'
                )
            );

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_menu_items_name_trgm
    ON menu_items USING GIN (name gin_trgm_ops);

-- +goose Down

DROP INDEX idx_menu_items_name_trgm;

UPDATE orders
SET status = 'ready'
WHERE status = 'delivered';

UPDATE orders
SET status = 'rejected'
WHERE status = 'cancelled';

ALTER TABLE orders
    DROP CONSTRAINT orders_status_check;

ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
        CHECK (
            status IN (
                       'created',
                       'accepted',
                       'preparing',
                       'ready',
                       'rejected'
                )
            );
