-- +goose Up
-- +goose StatementBegin
CREATE TABLE "customers"
(
    "id"                uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"        timestamptz       NOT NULL DEFAULT NOW(),
    "updated_at"        timestamptz       NOT NULL DEFAULT NOW(),
    "tg_id"             bigint            NOT NULL,
    "tg_username"       character varying NULL,
    "tg_firstname"      character varying NULL,
    "tg_lastname"       character varying NULL,
    "tg_language"       character varying NULL,
    "tg_picture"        character varying NULL,
    "tg_is_premium"     boolean           NOT NULL DEFAULT FALSE,
    "tg_is_allow_pm"    boolean           NOT NULL DEFAULT FALSE,
    "referrer_tg_id"    bigint            NULL,
    "wallet_hd_id"      bigint            GENERATED ALWAYS AS IDENTITY,
    "address_bounceable"    character varying NULL,
    "address_nonbounceable" character varying NULL,
    "ton_balance"       double precision  NOT NULL DEFAULT 0,
    "ton_balance_locked" double precision NOT NULL DEFAULT 0,
    "status"            integer           NOT NULL DEFAULT 0,
    PRIMARY KEY ("id"),
    CONSTRAINT "customers_tg_id_unique" UNIQUE ("tg_id"),
    CONSTRAINT "customers_wallet_hd_id_unique" UNIQUE ("wallet_hd_id"),
    CONSTRAINT "customers_address_bounceable_unique" UNIQUE ("address_bounceable"),
    CONSTRAINT "customers_address_nonbounceable_unique" UNIQUE ("address_nonbounceable")
);

CREATE INDEX "idx_customers_address_bounceable" ON "customers" ("address_bounceable");
CREATE INDEX "idx_customers_address_nonbounceable" ON "customers" ("address_nonbounceable");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "customers" CASCADE;
-- +goose StatementEnd
