CREATE TABLE IF NOT EXISTS sessions (
    id uuid PRIMARY KEY,
    identity_id uuid NOT NULL REFERENCES identities ON DELETE CASCADE,
    refresh_token_hash bytea NOT NULL UNIQUE,
    created_at timestamp(0) with time zone NOT NULL,
    expires_at timestamp(0) with time zone NOT NULL,
    last_used_at timestamp(0) with time zone NOT NULL
);
