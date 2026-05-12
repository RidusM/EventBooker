CREATE TABLE IF NOT EXISTS bookings (
    id         UUID        PRIMARY KEY,
    event_id   UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    status     TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bookings_expiry
    ON bookings (status, expires_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_bookings_event_status
    ON bookings (event_id, status);