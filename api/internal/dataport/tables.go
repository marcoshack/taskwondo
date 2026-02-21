package dataport

// ExportTables defines all tables in FK-dependency order.
// This order is used for both export and import.
var ExportTables = []TableDef{
	{
		Name:     "users",
		Filename: "data/01_users.json",
		ExportQuery: `SELECT id, email, display_name, password_hash, global_role,
			avatar_url, is_active, last_login_at, created_at, updated_at
			FROM users ORDER BY created_at`,
		ImportColumns: []string{
			"id", "email", "display_name", "password_hash", "global_role",
			"avatar_url", "is_active", "last_login_at", "created_at", "updated_at",
		},
	},
	{
		Name:     "user_oauth_accounts",
		Filename: "data/02_user_oauth_accounts.json",
		ExportQuery: `SELECT id, user_id, provider, provider_user_id, provider_email,
			provider_username, provider_avatar, created_at, updated_at
			FROM user_oauth_accounts ORDER BY created_at`,
		ImportColumns: []string{
			"id", "user_id", "provider", "provider_user_id", "provider_email",
			"provider_username", "provider_avatar", "created_at", "updated_at",
		},
	},
	{
		Name:     "api_keys",
		Filename: "data/03_api_keys.json",
		ExportQuery: `SELECT id, user_id, name, key_hash, key_prefix, permissions,
			last_used_at, expires_at, created_at
			FROM api_keys ORDER BY created_at`,
		ImportColumns: []string{
			"id", "user_id", "name", "key_hash", "key_prefix", "permissions",
			"last_used_at", "expires_at", "created_at",
		},
	},
	{
		Name:     "workflows",
		Filename: "data/04_workflows.json",
		ExportQuery: `SELECT id, name, description, is_default, created_at, updated_at
			FROM workflows ORDER BY created_at`,
		ImportColumns: []string{
			"id", "name", "description", "is_default", "created_at", "updated_at",
		},
	},
	{
		Name:     "workflow_statuses",
		Filename: "data/05_workflow_statuses.json",
		ExportQuery: `SELECT id, workflow_id, name, display_name, category, position, color
			FROM workflow_statuses ORDER BY workflow_id, position`,
		ImportColumns: []string{
			"id", "workflow_id", "name", "display_name", "category", "position", "color",
		},
	},
	{
		Name:     "workflow_transitions",
		Filename: "data/06_workflow_transitions.json",
		ExportQuery: `SELECT id, workflow_id, from_status, to_status, name
			FROM workflow_transitions ORDER BY workflow_id`,
		ImportColumns: []string{
			"id", "workflow_id", "from_status", "to_status", "name",
		},
	},
	{
		Name:     "projects",
		Filename: "data/07_projects.json",
		ExportQuery: `SELECT id, name, key, description, item_counter,
			default_workflow_id, created_at, updated_at, deleted_at
			FROM projects ORDER BY created_at`,
		ImportColumns: []string{
			"id", "name", "key", "description", "item_counter",
			"default_workflow_id", "created_at", "updated_at", "deleted_at",
		},
	},
	{
		Name:     "project_members",
		Filename: "data/08_project_members.json",
		ExportQuery: `SELECT id, project_id, user_id, role, created_at
			FROM project_members ORDER BY created_at`,
		ImportColumns: []string{
			"id", "project_id", "user_id", "role", "created_at",
		},
	},
	{
		Name:     "queues",
		Filename: "data/09_queues.json",
		ExportQuery: `SELECT id, project_id, name, description, queue_type, is_public,
			default_priority, default_assignee_id, workflow_id, created_at, updated_at
			FROM queues ORDER BY created_at`,
		ImportColumns: []string{
			"id", "project_id", "name", "description", "queue_type", "is_public",
			"default_priority", "default_assignee_id", "workflow_id", "created_at", "updated_at",
		},
	},
	{
		Name:     "milestones",
		Filename: "data/10_milestones.json",
		ExportQuery: `SELECT id, project_id, name, description, due_date, status,
			created_at, updated_at
			FROM milestones ORDER BY created_at`,
		ImportColumns: []string{
			"id", "project_id", "name", "description", "due_date", "status",
			"created_at", "updated_at",
		},
	},
	{
		Name:     "work_items",
		Filename: "data/11_work_items.json",
		// Excludes search_vector (GENERATED ALWAYS column).
		ExportQuery: `SELECT id, project_id, queue_id, parent_id, item_number,
			display_id, type, title, description, status, priority,
			assignee_id, reporter_id, portal_contact_id, visibility,
			labels, custom_fields, due_date, resolved_at,
			milestone_id, created_at, updated_at, deleted_at
			FROM work_items ORDER BY created_at`,
		ImportColumns: []string{
			"id", "project_id", "queue_id", "parent_id", "item_number",
			"display_id", "type", "title", "description", "status", "priority",
			"assignee_id", "reporter_id", "portal_contact_id", "visibility",
			"labels", "custom_fields", "due_date", "resolved_at",
			"milestone_id", "created_at", "updated_at", "deleted_at",
		},
	},
	{
		Name:     "work_item_events",
		Filename: "data/12_work_item_events.json",
		ExportQuery: `SELECT id, work_item_id, actor_id, event_type, field_name,
			old_value, new_value, metadata, visibility, created_at
			FROM work_item_events ORDER BY created_at`,
		ImportColumns: []string{
			"id", "work_item_id", "actor_id", "event_type", "field_name",
			"old_value", "new_value", "metadata", "visibility", "created_at",
		},
	},
	{
		Name:     "comments",
		Filename: "data/13_comments.json",
		ExportQuery: `SELECT id, work_item_id, author_id, portal_contact_id, body,
			visibility, edit_count, created_at, updated_at, deleted_at
			FROM comments ORDER BY created_at`,
		ImportColumns: []string{
			"id", "work_item_id", "author_id", "portal_contact_id", "body",
			"visibility", "edit_count", "created_at", "updated_at", "deleted_at",
		},
	},
	{
		Name:     "work_item_relations",
		Filename: "data/14_work_item_relations.json",
		ExportQuery: `SELECT id, source_id, target_id, relation_type, created_by, created_at
			FROM work_item_relations ORDER BY created_at`,
		ImportColumns: []string{
			"id", "source_id", "target_id", "relation_type", "created_by", "created_at",
		},
	},
	{
		Name:     "attachments",
		Filename: "data/15_attachments.json",
		ExportQuery: `SELECT id, work_item_id, uploader_id, filename, content_type,
			size_bytes, storage_key, comment, created_at, deleted_at
			FROM attachments ORDER BY created_at`,
		ImportColumns: []string{
			"id", "work_item_id", "uploader_id", "filename", "content_type",
			"size_bytes", "storage_key", "comment", "created_at", "deleted_at",
		},
	},
	{
		Name:     "user_settings",
		Filename: "data/16_user_settings.json",
		ExportQuery: `SELECT user_id, project_id, key, value, updated_at
			FROM user_settings ORDER BY user_id, key`,
		ImportColumns: []string{
			"user_id", "project_id", "key", "value", "updated_at",
		},
	},
}
