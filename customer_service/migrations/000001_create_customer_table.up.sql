CREATE TABLE IF NOT EXISTS customers (
    id uuid PRIMARY KEY,

    email text UNIQUE,
    email_verified bool NOT NULL DEFAULT false,

    first_name text,
    last_name text,
    middle_name text,
    birth_date date,
    gender text,

    profile_status text NOT NULL DEFAULT false,
    kyc_status text NOT NULL,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
