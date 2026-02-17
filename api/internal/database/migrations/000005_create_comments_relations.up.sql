-- Phase 5: Comments and Work Item Relations

-- comments table
CREATE TABLE comments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_item_id      UUID NOT NULL REFERENCES work_items(id),
    author_id         UUID REFERENCES users(id),
    portal_contact_id UUID,
    body              TEXT NOT NULL,
    visibility        TEXT NOT NULL DEFAULT 'internal'
                      CHECK (visibility IN ('internal', 'public')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX idx_comments_work_item ON comments(work_item_id, created_at)
    WHERE deleted_at IS NULL;

-- work_item_relations table
CREATE TABLE work_item_relations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id       UUID NOT NULL REFERENCES work_items(id),
    target_id       UUID NOT NULL REFERENCES work_items(id),
    relation_type   TEXT NOT NULL CHECK (relation_type IN (
                        'blocks', 'blocked_by', 'relates_to',
                        'duplicates', 'caused_by', 'parent_of', 'child_of')),
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (source_id, target_id, relation_type),
    CHECK (source_id != target_id)
);

CREATE INDEX idx_relations_source ON work_item_relations(source_id);
CREATE INDEX idx_relations_target ON work_item_relations(target_id);
