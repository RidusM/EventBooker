CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY ,
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    telegram_id BIGINT      NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);