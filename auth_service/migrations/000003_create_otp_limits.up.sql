CREATE TABLE IF NOT EXISTS otp_limits (
    destination text NOT NULL,
    channel text NOT NULL,
    purpose text NOT NULL,
    last_issued_at timestamptz NULL,
    issue_count integer NOT NULL DEFAULT 0 CHECK (issue_count >= 0),
    updated_at timestamptz NOT NULL DEFAULT NOW(),

    PRIMARY KEY (destination, channel, purpose)
);
