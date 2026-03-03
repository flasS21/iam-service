-- 0003_user_status_enum.sql

-- Create enum type (safe)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type WHERE typname = 'user_status'
    ) THEN
        CREATE TYPE user_status AS ENUM (
            'active',
            'disabled'
        );
    END IF;
END$$;

-- Drop default first (important!)
ALTER TABLE public.users
    ALTER COLUMN status DROP DEFAULT;

-- Convert text → enum
ALTER TABLE public.users
    ALTER COLUMN status TYPE user_status
    USING status::user_status;

-- Re-add correct enum default
ALTER TABLE public.users
    ALTER COLUMN status SET DEFAULT 'active';