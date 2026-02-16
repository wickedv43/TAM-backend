-- +goose Up
-- +goose StatementBegin
CREATE TABLE "transactions"
(
    "id"             uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"     timestamptz       NOT NULL DEFAULT NOW(),
    "updated_at"     timestamptz       NOT NULL DEFAULT NOW(),
    "sender"         character varying NOT NULL,
    "wallet_address" character varying NOT NULL,
    "amount"         double precision  NOT NULL,
    "tx_hash"        character varying NOT NULL,
    "seqno"          bigint            NOT NULL,
    "lt"             bigint            NOT NULL,
    "status"         integer           NOT NULL DEFAULT 0,
    PRIMARY KEY ("id"),
    CONSTRAINT "transactions_tx_hash_unique" UNIQUE ("tx_hash")
);

CREATE INDEX "idx_transactions_wallet_address" ON "transactions" ("wallet_address");
CREATE INDEX "idx_transactions_status" ON "transactions" ("status");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "transactions" CASCADE;
-- +goose StatementEnd
