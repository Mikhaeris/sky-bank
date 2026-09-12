CREATE TABLE IF NOT EXISTS tokens (
    hash bytea PRIMARY KEY,
    identities_id uuid NOT NULL REFERENCES identities ON DELETE CASCADE,
    expiry timestamp(0) with time zone NOT NULL,
    scope text NOT NULL
);
