-- +goose Up
-- +goose StatementBegin
CREATE TABLE "customers_briefs"
(
    "id"          uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"  timestamptz NOT NULL DEFAULT NOW(),
    "updated_at"  timestamptz NOT NULL DEFAULT NOW(),
    "customer_id" uuid        NOT NULL,
    "brief_id"    uuid        NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_customers_briefs_customer" FOREIGN KEY ("customer_id") 
        REFERENCES "customers" ("id") ON DELETE CASCADE,
    CONSTRAINT "fk_customers_briefs_brief" FOREIGN KEY ("brief_id") 
        REFERENCES "briefs" ("id") ON DELETE CASCADE
);

CREATE INDEX "idx_customers_briefs_customer_id" ON "customers_briefs" ("customer_id");
CREATE INDEX "idx_customers_briefs_brief_id" ON "customers_briefs" ("brief_id");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "customers_briefs" CASCADE;
-- +goose StatementEnd
