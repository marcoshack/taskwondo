DO $$
BEGIN
    IF to_regclass('embeddings') IS NOT NULL THEN
        ALTER TABLE embeddings DROP COLUMN IF EXISTS path;
    END IF;
END
$$;
