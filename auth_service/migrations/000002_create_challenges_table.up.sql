CREATE TABLE IF NOT EXISTS challenges (
    id uuid PRIMARY KEY,
    destination text NOT NULL,
    channel text NOT NULL,
    purpose text NOT NULL,
    code_hash bytea NOT NULL,
    expires_at timestamp(0) with time zone NOT NULL,
    failed_attempts integer NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),

    UNIQUE (destination, channel, purpose)
);
