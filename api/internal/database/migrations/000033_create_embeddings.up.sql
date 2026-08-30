DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'vector') THEN
        EXECUTE 'CREATE EXTENSION IF NOT EXISTS vector';
        EXECUTE 'CREATE TABLE embeddings (
            id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            entity_type TEXT NOT NULL,
            entity_id   UUID NOT NULL,
            project_id  UUID REFERENCES projects(id) ON DELETE CASCADE,
            content     TEXT NOT NULL,
            embedding   vector(768) NOT NULL,
            indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
            UNIQUE (entity_type, entity_id)
        )';
        EXECUTE 'CREATE INDEX idx_embeddings_vector ON embeddings USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 128)';
        EXECUTE 'CREATE INDEX idx_embeddings_project ON embeddings(project_id, entity_type) WHERE project_id IS NOT NULL';
        EXECUTE 'CREATE INDEX idx_embeddings_entity ON embeddings(entity_type, entity_id)';
    END IF;
END
$$;
