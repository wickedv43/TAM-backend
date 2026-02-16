-- +goose Up
-- +goose StatementBegin
CREATE TABLE "deals"
(
    "id"                     uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"             timestamptz       NOT NULL DEFAULT NOW(),
    "updated_at"             timestamptz       NOT NULL DEFAULT NOW(),
    "channel_id"             uuid              NOT NULL,
    "advertiser_customer_id" uuid              NOT NULL,
    "channel_manager_id"     uuid              NOT NULL,
    "type"                   integer           NOT NULL,
    "status"                 integer           NOT NULL DEFAULT 0,
    "status_updated_at"      timestamptz       NOT NULL DEFAULT NOW(),
    "expires_at"             timestamptz       NOT NULL,
    "target_type"            varchar(20)       NOT NULL DEFAULT 'post_1_24' CHECK ("target_type" IN ('post_1_24', 'post_2_48', 'post_3_72')),
    "target"                 jsonb             NOT NULL DEFAULT '{}',
    "publication_time"       timestamptz       NULL,
    "top_deadline"           timestamptz       NULL,
    "common_deadline"        timestamptz       NULL,
    "ton_price"              double precision  NOT NULL,
    "balance_refunded"       bool              NOT NULL DEFAULT FALSE,
    "channel_post_ids"       jsonb             DEFAULT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_deals_channel" FOREIGN KEY ("channel_id") 
        REFERENCES "channels" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_deals_advertiser" FOREIGN KEY ("advertiser_customer_id") 
        REFERENCES "customers" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_deals_channel_manager" FOREIGN KEY ("channel_manager_id") 
        REFERENCES "customers" ("id") ON DELETE CASCADE,
    CONSTRAINT "chk_deals_ton_price_positive" CHECK ("ton_price" > 0)
);

CREATE INDEX "idx_deals_channel_id" ON "deals" ("channel_id");
CREATE INDEX "idx_deals_advertiser_customer_id" ON "deals" ("advertiser_customer_id");
CREATE INDEX "idx_deals_channel_manager_id" ON "deals" ("channel_manager_id");
CREATE INDEX "idx_deals_status" ON "deals" ("status");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "deals" CASCADE;
-- +goose StatementEnd
