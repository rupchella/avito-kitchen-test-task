-- +goose Up

INSERT INTO restaurants (id, name, status)
VALUES (
    'a1b2c3d4-0000-4000-8000-000000000001',
    'AvitoCookRestaurant',
    'active'
);

INSERT INTO restaurant_integrations (restaurant_id, external_id, api_key_hash)
VALUES (
    'a1b2c3d4-0000-4000-8000-000000000001',
    'demo-restaurant',
    '4ed712d28db8409911fff6151f226c3ca6309e770b3df44f63e62100d4e29543'
);

-- +goose Down

DELETE FROM restaurant_integrations
WHERE restaurant_id = 'a1b2c3d4-0000-4000-8000-000000000001';

DELETE FROM restaurants
WHERE id = 'a1b2c3d4-0000-4000-8000-000000000001';
