CREATE TABLE IF NOT EXISTS otp_limits (
    destination text NOT NULL,
    channel text NOT NULL,
    purpose text NOT NULL,
    last_issued_at timestamptz NULL,
    issue_window_started_at timestamptz NULL,
    issue_count integer NOT NULL DEFAULT 0 CHECK (issue_count >= 0),
    failure_window_started_at timestamptz NULL,
    failure_count integer NOT NULL DEFAULT 0 CHECK (failure_count >= 0),
    blocked_until timestamptz NULL,
    updated_at timestamptz NOT NULL DEFAULT NOW(),

    PRIMARY KEY (destination, channel, purpose)
);
