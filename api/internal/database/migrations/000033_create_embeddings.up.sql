CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    project_id  UUID REFERENCES projects(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    embedding   vector(768) NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_type, entity_id)
);

CREATE INDEX idx_embeddings_vector ON embeddings
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 128);
CREATE INDEX idx_embeddings_project ON embeddings(project_id, entity_type) WHERE project_id IS NOT NULL;
CREATE INDEX idx_embeddings_entity ON embeddings(entity_type, entity_id);
