UPDATE embeddings SET project_id = entity_id WHERE entity_type = 'project' AND project_id IS NULL;
