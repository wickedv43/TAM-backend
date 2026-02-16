-- +goose Up
-- +goose StatementBegin
CREATE TABLE "customers_channels"
(
    "id"          uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"  timestamptz       NOT NULL DEFAULT NOW(),
    "updated_at"  timestamptz       NOT NULL DEFAULT NOW(),
    "customer_id" uuid              NOT NULL,
    "channel_id"  uuid              NOT NULL,
    "role"        integer           NOT NULL DEFAULT 0,
    "permissions" integer[]         NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_customers_channels_customer" FOREIGN KEY ("customer_id") 
        REFERENCES "customers" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_customers_channels_channel" FOREIGN KEY ("channel_id") 
        REFERENCES "channels" ("id") ON DELETE CASCADE
);

CREATE INDEX "idx_customers_channels_customer_id" ON "customers_channels" ("customer_id");
CREATE INDEX "idx_customers_channels_channel_id" ON "customers_channels" ("channel_id");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "customers_channels" CASCADE;
-- +goose StatementEnd
