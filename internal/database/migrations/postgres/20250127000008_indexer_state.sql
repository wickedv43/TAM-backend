-- +goose Up
-- +goose StatementBegin
CREATE TABLE "indexer_state"
(
    "id"           uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "service_name" character varying NOT NULL,
    "last_seqno"   bigint            NOT NULL DEFAULT 0,
    PRIMARY KEY ("id"),
    CONSTRAINT "indexer_state_service_name_unique" UNIQUE ("service_name")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "indexer_state" CASCADE;
-- +goose StatementEnd
