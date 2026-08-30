DO $$
BEGIN
    IF to_regclass('embeddings') IS NOT NULL THEN
        ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS path TEXT NOT NULL DEFAULT '';
    END IF;
END
$$;
