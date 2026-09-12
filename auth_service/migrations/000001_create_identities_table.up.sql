CREATE TABLE IF NOT EXISTS identities  (
    id uuid PRIMARY KEY,
    email text NOT NULL UNIQUE,
    password_hash bytea NOT NULL,
    activated bool NOT NULL,
    version integer NOT NULL DEFAULT 1
);
