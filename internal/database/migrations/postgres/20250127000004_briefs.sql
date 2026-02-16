-- +goose Up
-- +goose StatementBegin
CREATE TABLE "briefs"
(
    "id"         uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "created_at" timestamptz NOT NULL DEFAULT NOW(),
    "updated_at" timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY ("id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "briefs" CASCADE;
-- +goose StatementEnd
