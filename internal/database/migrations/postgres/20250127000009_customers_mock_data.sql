-- +goose Up
-- +goose StatementBegin
INSERT INTO customers (id, created_at, updated_at, tg_id, tg_username, tg_firstname, tg_lastname, tg_language, tg_picture, tg_is_premium, tg_is_allow_pm, referrer_tg_id, address_bounceable, address_nonbounceable, ton_balance, ton_balance_locked, status)
VALUES ('0dab2e71-2019-4a65-8739-6ed5b70d744e', '2025-01-27 12:00:00+00', '2025-01-27 12:00:00+00', 1, 'test_user_1', 'Test1', 'User1', 'en', NULL, false, true, NULL, 'EQAg9eG5-70qm0dYDaTbEm6U8tcNhGTfiY4Mx6AOAzMsxVyp', 'UQAg9eG5-70qm0dYDaTbEm6U8tcNhGTfiY4Mx6AOAzMsxQFs', 0, 0, 0);
INSERT INTO customers (id, created_at, updated_at, tg_id, tg_username, tg_firstname, tg_lastname, tg_language, tg_picture, tg_is_premium, tg_is_allow_pm, referrer_tg_id, address_bounceable, address_nonbounceable, ton_balance, ton_balance_locked, status)
VALUES ('cfdc7a17-009f-4e99-8212-fa14f446b3ab', '2025-01-27 12:00:00+00', '2025-01-27 12:00:00+00', 2, 'test_user_2', 'Test2', 'User2', 'ru', NULL, true, true, NULL, 'EQC1pzdWqGaS0G3G8jKN0SIZcsAzbMy4at771eff9sKLBVun', 'UQC1pzdWqGaS0G3G8jKN0SIZcsAzbMy4at771eff9sKLBQZi', 0, 0, 0);
INSERT INTO customers (id, created_at, updated_at, tg_id, tg_username, tg_firstname, tg_lastname, tg_language, tg_picture, tg_is_premium, tg_is_allow_pm, referrer_tg_id, address_bounceable, address_nonbounceable, ton_balance, ton_balance_locked, status)
VALUES ('80df3a0e-afec-4855-b529-7b4e5586f7a5', '2025-01-27 12:00:00+00', '2025-01-27 12:00:00+00', 3, 'test_user_3', 'Test3', 'User3', 'en', NULL, false, false, NULL, 'EQCwAQhGmc0aSfQM-nIcisbEugaJ_U8eoTffhnocwVkzVKsG', 'UQCwAQhGmc0aSfQM-nIcisbEugaJ_U8eoTffhnocwVkzVPbD', 0, 0, 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
TRUNCATE customers CASCADE;
-- +goose StatementEnd
