CREATE TABLE IF NOT EXISTS events (
    id              UUID        PRIMARY KEY,
    title           TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    date            TIMESTAMPTZ NOT NULL,
    total_seats     INT         NOT NULL CHECK (total_seats > 0),
    booking_ttl_min INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);