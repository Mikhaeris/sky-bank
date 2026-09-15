CREATE TABLE IF NOT EXISTS one_time_codes (
    email text NOT NULL,
    purpose text NOT NULL,
    code_hash bytea NOT NULL,
    expires_at timestamp(0) with time zone NOT NULL,

    PRIMARY KEY (email, purpose)
);
