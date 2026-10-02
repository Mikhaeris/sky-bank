CREATE TABLE IF NOT EXISTS otp_outbox (
    event_id uuid PRIMARY KEY,
    challenge_id uuid NOT NULL UNIQUE,

    encrypted_event bytea NOT NULL,
    trace_parent text,
    trace_state text,

    expires_at timestamptz NOT NULL,
    leased_until timestamptz
);

CREATE INDEX IF NOT EXISTS otp_outbox_expires_at_idx
    ON otp_outbox (expires_at);
