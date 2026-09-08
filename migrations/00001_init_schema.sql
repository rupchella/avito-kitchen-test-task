-- +goose Up
CREATE TABLE restaurants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT restaurants_name_check
        CHECK (CHAR_LENGTH(BTRIM(name)) BETWEEN 1 AND 200),

    CONSTRAINT restaurants_status_check
        CHECK ( status IN ('active', 'inactive'))

);

CREATE TABLE restaurant_integrations (
    restaurant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL,
    api_key_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT restaurant_integrations_restaurant_fk
        FOREIGN KEY (restaurant_id)
            REFERENCES restaurants (id)
            ON DELETE CASCADE,

    CONSTRAINT restaurant_integrations_external_id_key
        UNIQUE (external_id),

    CONSTRAINT restaurant_integrations_api_key_hash_key
        UNIQUE (api_key_hash),

    CONSTRAINT restaurant_integrations_external_id_check
        CHECK (CHAR_LENGTH(BTRIM(external_id)) BETWEEN 1 AND 100),

    CONSTRAINT restaurant_integrations_api_key_hash_check
        CHECK (CHAR_LENGTH(api_key_hash) = 64)
);

CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_kopecks BIGINT NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT menu_items_restaurant_fk
        FOREIGN KEY (restaurant_id)
            REFERENCES restaurants (id)
            ON DELETE RESTRICT,

    CONSTRAINT menu_items_restaurant_external_id_key
        UNIQUE (restaurant_id, external_id),

    CONSTRAINT menu_items_external_id_check
        CHECK (BTRIM(external_id) <> ''),

    CONSTRAINT menu_items_name_check
        CHECK (BTRIM(name) <> ''),

    CONSTRAINT menu_items_price_check
        CHECK (price_kopecks > 0)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created',
    total_price_kopecks BIGINT NOT NULL,
    delivery_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT orders_restaurant_fk
        FOREIGN KEY (restaurant_id)
            REFERENCES restaurants (id)
            ON DELETE RESTRICT,

    CONSTRAINT orders_user_idempotency_key
        UNIQUE (user_id, idempotency_key),

    CONSTRAINT orders_user_id_check
        CHECK (BTRIM(user_id) <> ''),

    CONSTRAINT orders_idempotency_key_check
        CHECK (BTRIM(idempotency_key) <> ''),

    CONSTRAINT orders_status_check
        CHECK (
            status IN (
                       'created',
                       'accepted',
                       'preparing',
                       'ready',
                       'rejected'
                )
            ),

    CONSTRAINT orders_total_price_check
        CHECK (total_price_kopecks > 0),

    CONSTRAINT orders_delivery_address_check
        CHECK (BTRIM(delivery_address) <> '')
);

CREATE TABLE order_items (
    order_id UUID NOT NULL,
    menu_item_id UUID NOT NULL,
    item_name TEXT NOT NULL,
    unit_price_kopecks BIGINT NOT NULL,
    quantity INTEGER NOT NULL,

    CONSTRAINT order_items_pk
        PRIMARY KEY (order_id, menu_item_id),

    CONSTRAINT order_items_order_fk
        FOREIGN KEY (order_id)
            REFERENCES orders (id)
            ON DELETE CASCADE,

    CONSTRAINT order_items_menu_item_fk
        FOREIGN KEY (menu_item_id)
            REFERENCES menu_items (id)
            ON DELETE RESTRICT,

    CONSTRAINT order_items_name_check
        CHECK (BTRIM(item_name) <> ''),

    CONSTRAINT order_items_unit_price_check
        CHECK (unit_price_kopecks > 0),

    CONSTRAINT order_items_quantity_check
        CHECK (quantity > 0)
);

-- +goose Down

DROP TABLE order_items;
DROP TABLE orders;
DROP TABLE menu_items;
DROP TABLE restaurant_integrations;
DROP TABLE restaurants;
