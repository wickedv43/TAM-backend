-- +goose Up
-- +goose StatementBegin
INSERT INTO channels (id, created_at, updated_at, tg_id, tg_username, tg_name, tg_description, tg_picture, is_listed, tags, stats_updated_at, stats, prices, subscribers, premium_subscribers, median_post_views, avg_post_views, status, main_language, restricted_from, restricted_till, restriction_reason)
VALUES (
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    '2025-02-03 12:00:00+00',
    '2025-02-03 12:00:00+00',
    1001,
    'test_channel',
    'Test Channel',
    'A test channel for development',
    NULL,
    true,
    '["tech", "test"]',
    '2025-02-03 12:00:00+00',
    '{}',
    '{"post_1_24": 10.0, "post_2_48": 8.0, "post_3_72": 5.0}',
    1000,
    50,
    500,
    450,
    1,
    'en',
    '1970-01-01 00:00:00+00',
    '1970-01-01 00:00:00+00',
    'unspecified'
);

-- Link test_user_1 as OWNER of the channel (role=1)
INSERT INTO customers_channels (id, created_at, updated_at, customer_id, channel_id, role, permissions)
VALUES (
    'b2c3d4e5-f6a7-8901-bcde-f12345678901',
    '2025-02-03 12:00:00+00',
    '2025-02-03 12:00:00+00',
    '0dab2e71-2019-4a65-8739-6ed5b70d744e',
    'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
    1,
    NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM customers_channels WHERE channel_id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
DELETE FROM channels WHERE id = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
-- +goose StatementEnd
