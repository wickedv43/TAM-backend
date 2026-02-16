-- +goose Up
-- +goose StatementBegin
CREATE TABLE "channels"
(
    "id"                  uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "created_at"          timestamptz       NOT NULL DEFAULT NOW(),
    "updated_at"          timestamptz       NOT NULL DEFAULT NOW(),
    "tg_id"               bigint            NOT NULL,
    "tg_username"         character varying NOT NULL,
    "tg_name"             character varying NULL,
    "tg_description"      character varying NULL,
    "tg_picture"          character varying NULL,
    "commentary"          text              NULL,
    "is_listed"           boolean           NOT NULL DEFAULT FALSE,
    "tags"                jsonb             NULL,
    "stats_updated_at"    timestamptz       NOT NULL DEFAULT NOW(),
    "stats"               jsonb             NULL,
    "prices"              jsonb             NULL,
    "subscribers"         integer           NULL,
    "premium_subscribers" integer           NULL,
    "median_post_views"   integer           NULL,
    "avg_post_views"      integer           NULL,
    "notifications_on"    float             NOT NULL DEFAULT 0,
    "first_post_date"     timestamptz       NULL,
    "total_posts"         integer           NULL,
    "status"              integer           NOT NULL DEFAULT 0,
    "main_language"       character varying NULL,
    "restricted_from"     timestamptz       NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    "restricted_till"     timestamptz       NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    "restriction_reason"  character varying NOT NULL DEFAULT 'unspecified',
    PRIMARY KEY ("id"),
    CONSTRAINT "channels_tg_id_unique" UNIQUE ("tg_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "channels" CASCADE;
-- +goose StatementEnd
