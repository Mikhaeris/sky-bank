CREATE TABLE IF NOT EXISTS identities  (
    id uuid PRIMARY KEY,
    email text NOT NULL UNIQUE,
    version integer NOT NULL DEFAULT 1
);
