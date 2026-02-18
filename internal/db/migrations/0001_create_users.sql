CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE IF NOT EXISTS users (

    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    keycloak_sub text NOT NULL UNIQUE,

    email citext NOT NULL UNIQUE,
    email_verified boolean NOT NULL DEFAULT false,

    status text NOT NULL DEFAULT 'active',

    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);
