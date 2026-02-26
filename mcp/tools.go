package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseDisplayID splits "TF-141" into ("TF", 141).
func parseDisplayID(displayID string) (string, int, error) {
	parts := strings.SplitN(displayID, "-", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid display_id format %q: expected PROJECT-NUMBER (e.g. TF-141)", displayID)
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid item number in display_id %q: %w", displayID, err)
	}
	return parts[0], num, nil
}

// getClient returns a configured API client, or an error tool result if not authenticated.
func getClient() (*Client, error) {
	baseURL, apiKey, err := ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w", err)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("TASKWONDO_URL not set. Set the environment variable or run the login tool first")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("not authenticated. Set TASKWONDO_API_KEY environment variable or run the login tool")
	}
	return NewClient(baseURL, apiKey), nil
}

// --- Tool definitions ---

func whoamiTool() mcp.Tool {
	return mcp.NewTool("whoami",
		mcp.WithDescription("Show the currently authenticated Taskwondo user and instance URL"),
	)
}

func loginTool() mcp.Tool {
	return mcp.NewTool("login",
		mcp.WithDescription("Log in to Taskwondo via browser-based authorization flow. Opens a browser window for the user to authorize access."),
	)
}

func logoutTool() mcp.Tool {
	return mcp.NewTool("logout",
		mcp.WithDescription("Log out from Taskwondo by clearing stored credentials"),
	)
}

func getWorkItemTool() mcp.Tool {
	return mcp.NewTool("get_work_item",
		mcp.WithDescription("Get a work item by its display ID (e.g. TF-141). Returns full details including title, description, status, priority, type, assignee, labels, and due date."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func listWorkItemsTool() mcp.Tool {
	return mcp.NewTool("list_work_items",
		mcp.WithDescription("List work items in a project with optional filters. IMPORTANT: When filtering by status, call list_statuses first to get the exact status names available in the project's workflow. Status names are system-wide but each project selects which ones to use, and they are case-sensitive (e.g. 'open', 'in_progress', not 'Open' or 'In Progress')."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithArray("status", mcp.WithStringItems(), mcp.Description("Filter by exact status names from list_statuses (case-sensitive, e.g. 'open', 'in_progress')")),
		mcp.WithArray("priority", mcp.WithStringItems(), mcp.Description("Filter by priority: critical, high, medium, low")),
		mcp.WithArray("type", mcp.WithStringItems(), mcp.Description("Filter by type: task, ticket, bug, feedback, epic")),
		mcp.WithString("assignee", mcp.Description("Filter by assignee: 'me', 'unassigned', or user email")),
		mcp.WithString("search", mcp.Description("Full-text search query")),
		mcp.WithNumber("limit", mcp.Description("Max results to return (default 20)")),
	)
}

func createWorkItemTool() mcp.Tool {
	return mcp.NewTool("create_work_item",
		mcp.WithDescription("Create a new work item in a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Work item title")),
		mcp.WithString("type", mcp.Description("Work item type: task, ticket, bug, feedback, epic (default: task)")),
		mcp.WithString("priority", mcp.Description("Priority: critical, high, medium, low (default: medium)")),
		mcp.WithString("description", mcp.Description("Work item description (markdown supported)")),
		mcp.WithArray("labels", mcp.WithStringItems(), mcp.Description("Labels to attach")),
		mcp.WithString("due_date", mcp.Description("Due date in YYYY-MM-DD format")),
	)
}

func updateWorkItemTool() mcp.Tool {
	return mcp.NewTool("update_work_item",
		mcp.WithDescription("Update a work item's fields. Only provided fields are changed."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("title", mcp.Description("New title")),
		mcp.WithString("status", mcp.Description("New status name from list_statuses (must be a valid transition from current status)")),
		mcp.WithString("priority", mcp.Description("New priority: critical, high, medium, low")),
		mcp.WithString("description", mcp.Description("New description (markdown supported)")),
		mcp.WithString("assignee", mcp.Description("Assignee user ID, or 'none' to unassign")),
		mcp.WithArray("labels", mcp.WithStringItems(), mcp.Description("New labels (replaces existing)")),
		mcp.WithString("due_date", mcp.Description("Due date YYYY-MM-DD, or 'none' to clear")),
		mcp.WithString("milestone_id", mcp.Description("Milestone UUID, or 'none' to clear")),
	)
}

func addCommentTool() mcp.Tool {
	return mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Comment body (markdown supported)")),
		mcp.WithBoolean("internal", mcp.Description("If true, comment is internal-only (not visible to external users). Default: false")),
	)
}

func listCommentsTool() mcp.Tool {
	return mcp.NewTool("list_comments",
		mcp.WithDescription("List comments on a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func listProjectsTool() mcp.Tool {
	return mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects the authenticated user has access to"),
	)
}

func getProjectTool() mcp.Tool {
	return mcp.NewTool("get_project",
		mcp.WithDescription("Get details of a specific project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func listStatusesTool() mcp.Tool {
	return mcp.NewTool("list_statuses",
		mcp.WithDescription("List available workflow statuses for a project, including their categories (todo, in_progress, done, cancelled). Call this before filtering work items by status to get the exact status names. Statuses are system-wide but each project selects which ones to use via its workflow."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func listEventsTool() mcp.Tool {
	return mcp.NewTool("list_events",
		mcp.WithDescription("List activity events on a work item. Shows status changes, field updates, comments added/removed, and other tracked changes."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func createRelationTool() mcp.Tool {
	return mcp.NewTool("create_relation",
		mcp.WithDescription("Create a relation between two work items. Use after creating related work items to link them together."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Source work item display ID, e.g. TF-141")),
		mcp.WithString("target_display_id", mcp.Required(), mcp.Description("Target work item display ID, e.g. TF-143")),
		mcp.WithString("relation_type", mcp.Required(), mcp.Description("Relation type: blocks, blocked_by, relates_to, duplicates, caused_by, parent_of, child_of")),
	)
}

func listRelationsTool() mcp.Tool {
	return mcp.NewTool("list_relations",
		mcp.WithDescription("List relations on a work item. Shows links to other work items (e.g. blocks, is blocked by, relates to)."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func listAttachmentsTool() mcp.Tool {
	return mcp.NewTool("list_attachments",
		mcp.WithDescription("List file attachments on a work item."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func uploadAttachmentTool() mcp.Tool {
	return mcp.NewTool("upload_attachment",
		mcp.WithDescription("Upload a file attachment to a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to the local file to upload")),
		mcp.WithString("comment", mcp.Description("Optional comment describing the attachment")),
	)
}

func logTimeTool() mcp.Tool {
	return mcp.NewTool("log_time",
		mcp.WithDescription("Log a time entry on a work item. Used to track time spent working on tasks."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithNumber("duration_seconds", mcp.Required(), mcp.Description("Duration in seconds (minimum 60)")),
		mcp.WithString("started_at", mcp.Description("When work started, in RFC3339 format (e.g. 2026-02-24T10:00:00Z). Defaults to now minus duration.")),
		mcp.WithString("description", mcp.Description("Brief description of the work done")),
	)
}

func listTimeEntriesTool() mcp.Tool {
	return mcp.NewTool("list_time_entries",
		mcp.WithDescription("List time entries on a work item. Returns individual entries and total logged seconds."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func updateTimeEntryTool() mcp.Tool {
	return mcp.NewTool("update_time_entry",
		mcp.WithDescription("Update an existing time entry. Only provided fields are changed."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("time_entry_id", mcp.Required(), mcp.Description("Time entry UUID to update")),
		mcp.WithNumber("duration_seconds", mcp.Description("New duration in seconds")),
		mcp.WithString("started_at", mcp.Description("New start time in RFC3339 format")),
		mcp.WithString("description", mcp.Description("New description, or empty string to clear")),
	)
}

func deleteTimeEntryTool() mcp.Tool {
	return mcp.NewTool("delete_time_entry",
		mcp.WithDescription("Delete a time entry from a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("time_entry_id", mcp.Required(), mcp.Description("Time entry UUID to delete")),
	)
}

// --- Tool handlers ---

func handleWhoami(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	user, err := client.GetMe()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get user info: %v", err)), nil
	}

	baseURL, _, _ := ResolveAuth()
	result := fmt.Sprintf("Logged in as: %s (%s)\nRole: %s\nInstance: %s",
		user.DisplayName, user.Email, user.GlobalRole, baseURL)
	return mcp.NewToolResultText(result), nil
}

func handleLogin(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	baseURL, _, _ := ResolveAuth()
	if baseURL == "" {
		return mcp.NewToolResultError("TASKWONDO_URL environment variable is required for login"), nil
	}

	creds, err := BrowserLogin(baseURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Login failed: %v", err)), nil
	}

	// Verify the key works
	client := NewClient(creds.URL, creds.APIKey)
	user, err := client.GetMe()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Login succeeded but verification failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully logged in as %s (%s)\nCredentials saved to %s",
		user.DisplayName, user.Email, ConfigDir())), nil
}

func handleLogout(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := DeleteCredentials(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clear credentials: %v", err)), nil
	}
	return mcp.NewToolResultText("Logged out. Credentials removed."), nil
}

func handleGetWorkItem(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	item, err := client.GetWorkItem(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get work item: %v", err)), nil
	}

	return mcp.NewToolResultText(formatWorkItemWithClient(item, client)), nil
}

func handleListWorkItems(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := ListWorkItemsParams{
		Project:    project,
		Statuses:   request.GetStringSlice("status", nil),
		Priorities: request.GetStringSlice("priority", nil),
		Types:      request.GetStringSlice("type", nil),
		Assignee:   request.GetString("assignee", ""),
		Search:     request.GetString("search", ""),
		Limit:      request.GetInt("limit", 20),
	}

	result, err := client.ListWorkItems(params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list work items: %v", err)), nil
	}

	if len(result.Items) == 0 {
		return mcp.NewToolResultText("No work items found matching the criteria."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d work items (total: %d):\n\n", len(result.Items), result.Total)
	for _, item := range result.Items {
		fmt.Fprintf(&sb, "- **%s** [%s] %s (status: %s, priority: %s)\n",
			item.DisplayID, item.Type, item.Title, item.Status, item.Priority)
	}
	if result.HasMore {
		sb.WriteString("\n(more results available — increase limit or refine filters)")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleCreateWorkItem(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := CreateWorkItemParams{
		Project:  project,
		Title:    title,
		Type:     request.GetString("type", ""),
		Priority: request.GetString("priority", ""),
		Labels:   request.GetStringSlice("labels", nil),
	}

	if desc := request.GetString("description", ""); desc != "" {
		params.Description = &desc
	}
	if dd := request.GetString("due_date", ""); dd != "" {
		params.DueDate = &dd
	}

	item, err := client.CreateWorkItem(params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created work item %s: %s\n\n%s",
		item.DisplayID, item.Title, formatWorkItemWithClient(item, client))), nil
}

func handleUpdateWorkItem(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates := make(map[string]interface{})

	if v := request.GetString("title", ""); v != "" {
		updates["title"] = v
	}
	if v := request.GetString("status", ""); v != "" {
		updates["status"] = v
	}
	if v := request.GetString("priority", ""); v != "" {
		updates["priority"] = v
	}
	if v := request.GetString("description", ""); v != "" {
		updates["description"] = v
	}
	if v := request.GetString("assignee", ""); v != "" {
		if v == "none" {
			updates["assignee_id"] = nil
		} else if v == "me" {
			me, err := client.GetMe()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to resolve 'me': %v", err)), nil
			}
			updates["assignee_id"] = me.ID
		} else {
			updates["assignee_id"] = v
		}
	}
	if labels := request.GetStringSlice("labels", nil); labels != nil {
		updates["labels"] = labels
	}
	if v := request.GetString("due_date", ""); v != "" {
		if v == "none" {
			updates["due_date"] = nil
		} else {
			updates["due_date"] = v
		}
	}
	if v := request.GetString("milestone_id", ""); v != "" {
		if v == "none" {
			updates["milestone_id"] = nil
		} else {
			updates["milestone_id"] = v
		}
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one field to change."), nil
	}

	item, err := client.UpdateWorkItem(projectKey, itemNumber, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated %s\n\n%s", item.DisplayID, formatWorkItemWithClient(item, client))), nil
}

func handleAddComment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body, err := request.RequireString("body")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := CreateCommentParams{
		Body: body,
	}
	if request.GetBool("internal", false) {
		params.Visibility = "internal"
	}

	comment, err := client.CreateComment(projectKey, itemNumber, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Comment added to %s (id: %s)", displayID, comment.ID)), nil
}

func handleListComments(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	comments, err := client.ListComments(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list comments: %v", err)), nil
	}

	if len(comments) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No comments on %s", displayID)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Comments on %s (%d):\n\n", displayID, len(comments))
	for _, c := range comments {
		vis := ""
		if c.Visibility == "internal" {
			vis = " [internal]"
		}
		fmt.Fprintf(&sb, "---\n**%s**%s\n%s\n\n", c.CreatedAt, vis, c.Body)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleListProjects(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projects, err := client.ListProjects()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	if len(projects) == 0 {
		return mcp.NewToolResultText("No projects found."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Projects (%d):\n\n", len(projects))
	for _, p := range projects {
		desc := ""
		if p.Description != nil && *p.Description != "" {
			desc = " — " + *p.Description
		}
		fmt.Fprintf(&sb, "- **%s** (%s)%s [%d open, %d in progress]\n",
			p.Key, p.Name, desc, p.OpenCount, p.InProgressCount)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleGetProject(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	p, err := client.GetProject(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project: %v", err)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s (%s)\n", p.Key, p.Name)
	if p.Description != nil && *p.Description != "" {
		fmt.Fprintf(&sb, "Description: %s\n", *p.Description)
	}
	fmt.Fprintf(&sb, "Items: %d total\n", p.ItemCounter)
	fmt.Fprintf(&sb, "Created: %s\n", p.CreatedAt)
	return mcp.NewToolResultText(sb.String()), nil
}

func handleListStatuses(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	statuses, err := client.ListProjectStatuses(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list statuses: %v", err)), nil
	}

	if len(statuses) == 0 {
		return mcp.NewToolResultText("No statuses found for this project."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Statuses for project %s:\n\n", project)
	for _, s := range statuses {
		fmt.Fprintf(&sb, "- **%s** (category: %s)\n", s.Name, s.Category)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleUploadAttachment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if _, err := os.Stat(filePath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("File not found: %v", err)), nil
	}

	comment := request.GetString("comment", "")

	attachment, err := client.UploadAttachment(projectKey, itemNumber, filePath, comment)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to upload attachment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Uploaded %s to %s (%s, %d bytes)\nID: %s\nDownload: %s",
		attachment.Filename, displayID, attachment.ContentType, attachment.SizeBytes,
		attachment.ID, attachment.DownloadURL)), nil
}

func handleListEvents(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	events, err := client.ListEvents(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list events: %v", err)), nil
	}

	if len(events) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No activity events on %s", displayID)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Activity on %s (%d events):\n\n", displayID, len(events))
	for _, e := range events {
		actor := "system"
		if e.Actor != nil && e.Actor.DisplayName != "" {
			actor = e.Actor.DisplayName
		}
		fmt.Fprintf(&sb, "- **%s** %s by %s", e.CreatedAt, e.EventType, actor)
		if e.FieldName != nil {
			old := "<empty>"
			if e.OldValue != nil {
				old = *e.OldValue
			}
			new := "<empty>"
			if e.NewValue != nil {
				new = *e.NewValue
			}
			fmt.Fprintf(&sb, " (%s: %s → %s)", *e.FieldName, old, new)
		}
		sb.WriteString("\n")
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleCreateRelation(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	targetDisplayID, err := request.RequireString("target_display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	relationType, err := request.RequireString("relation_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	relation, err := client.CreateRelation(projectKey, itemNumber, targetDisplayID, relationType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create relation: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created relation: %s **%s** %s",
		relation.SourceDisplayID, relation.RelationType, relation.TargetDisplayID)), nil
}

func handleListRelations(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	relations, err := client.ListRelations(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list relations: %v", err)), nil
	}

	if len(relations) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No relations on %s", displayID)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Relations on %s (%d):\n\n", displayID, len(relations))
	for _, r := range relations {
		fmt.Fprintf(&sb, "- %s **%s** %s (%s → %s)\n",
			r.SourceDisplayID, r.RelationType, r.TargetDisplayID,
			r.SourceTitle, r.TargetTitle)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleListAttachments(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	attachments, err := client.ListAttachments(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list attachments: %v", err)), nil
	}

	if len(attachments) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No attachments on %s", displayID)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Attachments on %s (%d):\n\n", displayID, len(attachments))
	for _, a := range attachments {
		comment := ""
		if a.Comment != "" {
			comment = fmt.Sprintf(" — %s", a.Comment)
		}
		fmt.Fprintf(&sb, "- **%s** (%s, %d bytes)%s\n  ID: %s | Download: %s\n",
			a.Filename, a.ContentType, a.SizeBytes, comment, a.ID, a.DownloadURL)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleLogTime(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	durationSeconds := request.GetInt("duration_seconds", 0)
	if durationSeconds < 60 {
		return mcp.NewToolResultError("duration_seconds must be at least 60"), nil
	}

	startedAt := request.GetString("started_at", "")
	if startedAt == "" {
		startedAt = time.Now().Add(-time.Duration(durationSeconds) * time.Second).UTC().Format(time.RFC3339)
	}

	params := CreateTimeEntryParams{
		StartedAt:       startedAt,
		DurationSeconds: durationSeconds,
		Description:     request.GetString("description", ""),
	}

	entry, err := client.CreateTimeEntry(projectKey, itemNumber, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to log time: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Logged %s on %s (id: %s)",
		formatDuration(entry.DurationSeconds), displayID, entry.ID)), nil
}

func handleListTimeEntries(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := client.ListTimeEntries(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list time entries: %v", err)), nil
	}

	if len(result.Entries) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No time entries on %s", displayID)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Time entries on %s (%d entries, total: %s):\n\n",
		displayID, len(result.Entries), formatDuration(result.TotalLoggedSeconds))
	for _, e := range result.Entries {
		desc := ""
		if e.Description != nil && *e.Description != "" {
			desc = fmt.Sprintf(" — %s", *e.Description)
		}
		fmt.Fprintf(&sb, "- **%s** %s (started: %s)%s\n  ID: %s\n",
			formatDuration(e.DurationSeconds), e.UserID, e.StartedAt, desc, e.ID)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleUpdateTimeEntry(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	entryID, err := request.RequireString("time_entry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates := make(map[string]interface{})
	if v := request.GetInt("duration_seconds", 0); v > 0 {
		updates["duration_seconds"] = v
	}
	if v := request.GetString("started_at", ""); v != "" {
		updates["started_at"] = v
	}
	// Allow setting description to empty string to clear it
	if desc, ok := request.GetArguments()["description"]; ok {
		updates["description"] = desc
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one of: duration_seconds, started_at, description."), nil
	}

	entry, err := client.UpdateTimeEntry(projectKey, itemNumber, entryID, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update time entry: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated time entry %s on %s (%s)",
		entry.ID, displayID, formatDuration(entry.DurationSeconds))), nil
}

func handleDeleteTimeEntry(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	entryID, err := request.RequireString("time_entry_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteTimeEntry(projectKey, itemNumber, entryID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete time entry: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted time entry %s from %s", entryID, displayID)), nil
}

// --- New tool definitions ---

func deleteWorkItemTool() mcp.Tool {
	return mcp.NewTool("delete_work_item",
		mcp.WithDescription("Delete a work item permanently"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func updateCommentTool() mcp.Tool {
	return mcp.NewTool("update_comment",
		mcp.WithDescription("Update the body of an existing comment on a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("comment_id", mcp.Required(), mcp.Description("Comment UUID to update")),
		mcp.WithString("body", mcp.Required(), mcp.Description("New comment body (markdown supported)")),
	)
}

func deleteCommentTool() mcp.Tool {
	return mcp.NewTool("delete_comment",
		mcp.WithDescription("Delete a comment from a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("comment_id", mcp.Required(), mcp.Description("Comment UUID to delete")),
	)
}

func deleteRelationTool() mcp.Tool {
	return mcp.NewTool("delete_relation",
		mcp.WithDescription("Delete a relation between work items"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("relation_id", mcp.Required(), mcp.Description("Relation UUID to delete (from list_relations)")),
	)
}

func downloadAttachmentTool() mcp.Tool {
	return mcp.NewTool("download_attachment",
		mcp.WithDescription("Download a file attachment from a work item to a local path"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("attachment_id", mcp.Required(), mcp.Description("Attachment UUID to download (from list_attachments)")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to save the downloaded file to")),
	)
}

func deleteAttachmentTool() mcp.Tool {
	return mcp.NewTool("delete_attachment",
		mcp.WithDescription("Delete a file attachment from a work item"),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
		mcp.WithString("attachment_id", mcp.Required(), mcp.Description("Attachment UUID to delete (from list_attachments)")),
	)
}

func createProjectTool() mcp.Tool {
	return mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Project display name")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Project key (2-10 uppercase letters/digits, must start with a letter, e.g. MYPROJ)")),
		mcp.WithString("description", mcp.Description("Optional project description")),
	)
}

func updateProjectTool() mcp.Tool {
	return mcp.NewTool("update_project",
		mcp.WithDescription("Update a project's fields. Only provided fields are changed."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("name", mcp.Description("New project name")),
		mcp.WithString("description", mcp.Description("New description, or 'none' to clear")),
	)
}

func deleteProjectTool() mcp.Tool {
	return mcp.NewTool("delete_project",
		mcp.WithDescription("Delete a project and all its work items. This is irreversible."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func listMembersTool() mcp.Tool {
	return mcp.NewTool("list_members",
		mcp.WithDescription("List members of a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func addMemberTool() mcp.Tool {
	return mcp.NewTool("add_member",
		mcp.WithDescription("Add a user to a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID to add (use search_users to find)")),
		mcp.WithString("role", mcp.Required(), mcp.Description("Member role: member, admin, owner")),
	)
}

func updateMemberRoleTool() mcp.Tool {
	return mcp.NewTool("update_member_role",
		mcp.WithDescription("Update a project member's role"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID")),
		mcp.WithString("role", mcp.Required(), mcp.Description("New role: member, admin, owner")),
	)
}

func removeMemberTool() mcp.Tool {
	return mcp.NewTool("remove_member",
		mcp.WithDescription("Remove a user from a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User UUID to remove")),
	)
}

func searchUsersTool() mcp.Tool {
	return mcp.NewTool("search_users",
		mcp.WithDescription("Search for users by name or email. Useful for finding user IDs when adding project members or assigning work items."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query (name or email)")),
	)
}

func listMilestonesTool() mcp.Tool {
	return mcp.NewTool("list_milestones",
		mcp.WithDescription("List milestones for a project, including progress counts (open/closed work items)"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func createMilestoneTool() mcp.Tool {
	return mcp.NewTool("create_milestone",
		mcp.WithDescription("Create a new milestone in a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Milestone name")),
		mcp.WithString("description", mcp.Description("Optional milestone description")),
		mcp.WithString("due_date", mcp.Description("Due date in YYYY-MM-DD format")),
	)
}

func getMilestoneTool() mcp.Tool {
	return mcp.NewTool("get_milestone",
		mcp.WithDescription("Get details of a specific milestone including progress counts"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("milestone_id", mcp.Required(), mcp.Description("Milestone UUID")),
	)
}

func updateMilestoneTool() mcp.Tool {
	return mcp.NewTool("update_milestone",
		mcp.WithDescription("Update a milestone's fields. Only provided fields are changed."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("milestone_id", mcp.Required(), mcp.Description("Milestone UUID")),
		mcp.WithString("name", mcp.Description("New milestone name")),
		mcp.WithString("description", mcp.Description("New description, or 'none' to clear")),
		mcp.WithString("due_date", mcp.Description("New due date YYYY-MM-DD, or 'none' to clear")),
		mcp.WithString("status", mcp.Description("New status: open or closed")),
	)
}

func deleteMilestoneTool() mcp.Tool {
	return mcp.NewTool("delete_milestone",
		mcp.WithDescription("Delete a milestone from a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("milestone_id", mcp.Required(), mcp.Description("Milestone UUID")),
	)
}

func listQueuesTool() mcp.Tool {
	return mcp.NewTool("list_queues",
		mcp.WithDescription("List queues for a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
	)
}

func createQueueTool() mcp.Tool {
	return mcp.NewTool("create_queue",
		mcp.WithDescription("Create a new queue in a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Queue name")),
		mcp.WithString("queue_type", mcp.Required(), mcp.Description("Queue type: support, alerts, feedback, general")),
		mcp.WithString("description", mcp.Description("Optional queue description")),
		mcp.WithBoolean("is_public", mcp.Description("Whether the queue is public (default: false)")),
		mcp.WithString("default_priority", mcp.Description("Default priority for items: critical, high, medium, low")),
	)
}

func getQueueTool() mcp.Tool {
	return mcp.NewTool("get_queue",
		mcp.WithDescription("Get details of a specific queue"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("queue_id", mcp.Required(), mcp.Description("Queue UUID")),
	)
}

func updateQueueTool() mcp.Tool {
	return mcp.NewTool("update_queue",
		mcp.WithDescription("Update a queue's fields. Only provided fields are changed."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("queue_id", mcp.Required(), mcp.Description("Queue UUID")),
		mcp.WithString("name", mcp.Description("New queue name")),
		mcp.WithString("description", mcp.Description("New description, or 'none' to clear")),
		mcp.WithString("queue_type", mcp.Description("New queue type: support, alerts, feedback, general")),
		mcp.WithString("default_priority", mcp.Description("New default priority: critical, high, medium, low")),
	)
}

func deleteQueueTool() mcp.Tool {
	return mcp.NewTool("delete_queue",
		mcp.WithDescription("Delete a queue from a project"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project key, e.g. TF")),
		mcp.WithString("queue_id", mcp.Required(), mcp.Description("Queue UUID")),
	)
}

// --- New tool handlers ---

func handleDeleteWorkItem(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteWorkItem(projectKey, itemNumber); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted work item %s", displayID)), nil
}

func handleUpdateComment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	commentID, err := request.RequireString("comment_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body, err := request.RequireString("body")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	comment, err := client.UpdateComment(projectKey, itemNumber, commentID, body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated comment %s on %s", comment.ID, displayID)), nil
}

func handleDeleteComment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	commentID, err := request.RequireString("comment_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteComment(projectKey, itemNumber, commentID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete comment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted comment %s from %s", commentID, displayID)), nil
}

func handleDeleteRelation(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	relationID, err := request.RequireString("relation_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteRelation(projectKey, itemNumber, relationID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete relation: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted relation %s from %s", relationID, displayID)), nil
}

func handleDownloadAttachment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	attachmentID, err := request.RequireString("attachment_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	n, err := client.DownloadAttachment(projectKey, itemNumber, attachmentID, filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to download attachment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Downloaded attachment %s from %s to %s (%d bytes)", attachmentID, displayID, filePath, n)), nil
}

func handleDeleteAttachment(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	attachmentID, err := request.RequireString("attachment_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteAttachment(projectKey, itemNumber, attachmentID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete attachment: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted attachment %s from %s", attachmentID, displayID)), nil
}

func handleCreateProject(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	key, err := request.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := CreateProjectParams{Name: name, Key: key}
	if desc := request.GetString("description", ""); desc != "" {
		params.Description = &desc
	}

	p, err := client.CreateProject(params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created project %s (%s)", p.Key, p.Name)), nil
}

func handleUpdateProject(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates := make(map[string]interface{})
	if v := request.GetString("name", ""); v != "" {
		updates["name"] = v
	}
	if v := request.GetString("description", ""); v != "" {
		if v == "none" {
			updates["description"] = nil
		} else {
			updates["description"] = v
		}
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one field to change."), nil
	}

	p, err := client.UpdateProject(project, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated project %s (%s)", p.Key, p.Name)), nil
}

func handleDeleteProject(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteProject(project); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete project: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted project %s", project)), nil
}

func handleListMembers(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	members, err := client.ListMembers(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list members: %v", err)), nil
	}

	if len(members) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No members in project %s", project)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Members of %s (%d):\n\n", project, len(members))
	for _, m := range members {
		fmt.Fprintf(&sb, "- **%s** (%s) — role: %s | id: %s\n", m.DisplayName, m.Email, m.Role, m.UserID)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleAddMember(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	userID, err := request.RequireString("user_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	role, err := request.RequireString("role")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	member, err := client.AddMember(project, AddMemberParams{UserID: userID, Role: role})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add member: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added %s (%s) to %s as %s", member.DisplayName, member.Email, project, member.Role)), nil
}

func handleUpdateMemberRole(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	userID, err := request.RequireString("user_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	role, err := request.RequireString("role")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	member, err := client.UpdateMemberRole(project, userID, role)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update member role: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated %s role to %s in %s", member.DisplayName, member.Role, project)), nil
}

func handleRemoveMember(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	userID, err := request.RequireString("user_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.RemoveMember(project, userID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove member: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed user %s from project %s", userID, project)), nil
}

func handleSearchUsers(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	users, err := client.SearchUsers(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search users: %v", err)), nil
	}

	if len(users) == 0 {
		return mcp.NewToolResultText("No users found matching the query."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Users matching %q (%d):\n\n", query, len(users))
	for _, u := range users {
		fmt.Fprintf(&sb, "- **%s** (%s) | id: %s\n", u.DisplayName, u.Email, u.ID)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleListMilestones(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	milestones, err := client.ListMilestones(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list milestones: %v", err)), nil
	}

	if len(milestones) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No milestones in project %s", project)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Milestones in %s (%d):\n\n", project, len(milestones))
	for _, m := range milestones {
		due := ""
		if m.DueDate != nil {
			due = fmt.Sprintf(" | due: %s", *m.DueDate)
		}
		fmt.Fprintf(&sb, "- **%s** [%s]%s — %d/%d closed | id: %s\n",
			m.Name, m.Status, due, m.ClosedCount, m.TotalCount, m.ID)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleCreateMilestone(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := CreateMilestoneParams{Name: name}
	if desc := request.GetString("description", ""); desc != "" {
		params.Description = &desc
	}
	if dd := request.GetString("due_date", ""); dd != "" {
		params.DueDate = &dd
	}

	m, err := client.CreateMilestone(project, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create milestone: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created milestone %q in %s (id: %s)", m.Name, project, m.ID)), nil
}

func handleGetMilestone(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	milestoneID, err := request.RequireString("milestone_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	m, err := client.GetMilestone(project, milestoneID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get milestone: %v", err)), nil
	}

	return mcp.NewToolResultText(formatMilestone(m)), nil
}

func handleUpdateMilestone(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	milestoneID, err := request.RequireString("milestone_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates := make(map[string]interface{})
	if v := request.GetString("name", ""); v != "" {
		updates["name"] = v
	}
	if v := request.GetString("description", ""); v != "" {
		if v == "none" {
			updates["description"] = nil
		} else {
			updates["description"] = v
		}
	}
	if v := request.GetString("due_date", ""); v != "" {
		if v == "none" {
			updates["due_date"] = nil
		} else {
			updates["due_date"] = v
		}
	}
	if v := request.GetString("status", ""); v != "" {
		updates["status"] = v
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one of: name, description, due_date, status."), nil
	}

	m, err := client.UpdateMilestone(project, milestoneID, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update milestone: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated milestone %q (id: %s)", m.Name, m.ID)), nil
}

func handleDeleteMilestone(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	milestoneID, err := request.RequireString("milestone_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteMilestone(project, milestoneID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete milestone: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted milestone %s from project %s", milestoneID, project)), nil
}

func handleListQueues(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queues, err := client.ListQueues(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list queues: %v", err)), nil
	}

	if len(queues) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No queues in project %s", project)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Queues in %s (%d):\n\n", project, len(queues))
	for _, q := range queues {
		public := ""
		if q.IsPublic {
			public = " [public]"
		}
		fmt.Fprintf(&sb, "- **%s** [%s]%s | id: %s\n", q.Name, q.QueueType, public, q.ID)
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func handleCreateQueue(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	queueType, err := request.RequireString("queue_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := CreateQueueParams{
		Name:            name,
		QueueType:       queueType,
		IsPublic:        request.GetBool("is_public", false),
		DefaultPriority: request.GetString("default_priority", ""),
	}
	if desc := request.GetString("description", ""); desc != "" {
		params.Description = &desc
	}

	q, err := client.CreateQueue(project, params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create queue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created queue %q in %s (id: %s)", q.Name, project, q.ID)), nil
}

func handleGetQueue(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	queueID, err := request.RequireString("queue_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	q, err := client.GetQueue(project, queueID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get queue: %v", err)), nil
	}

	return mcp.NewToolResultText(formatQueue(q)), nil
}

func handleUpdateQueue(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	queueID, err := request.RequireString("queue_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	updates := make(map[string]interface{})
	if v := request.GetString("name", ""); v != "" {
		updates["name"] = v
	}
	if v := request.GetString("description", ""); v != "" {
		if v == "none" {
			updates["description"] = nil
		} else {
			updates["description"] = v
		}
	}
	if v := request.GetString("queue_type", ""); v != "" {
		updates["queue_type"] = v
	}
	if v := request.GetString("default_priority", ""); v != "" {
		updates["default_priority"] = v
	}

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one of: name, description, queue_type, default_priority."), nil
	}

	q, err := client.UpdateQueue(project, queueID, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update queue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated queue %q (id: %s)", q.Name, q.ID)), nil
}

func handleDeleteQueue(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	project, err := request.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	queueID, err := request.RequireString("queue_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.DeleteQueue(project, queueID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete queue: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted queue %s from project %s", queueID, project)), nil
}

// --- Inbox tool definitions ---

func listInboxTool() mcp.Tool {
	return mcp.NewTool("list_inbox",
		mcp.WithDescription("List work items in the user's inbox. Returns items ordered by position with work item details including status, priority, and project info."),
		mcp.WithString("search", mcp.Description("Full-text search query")),
		mcp.WithString("include_completed", mcp.Description("Include completed items (default: false)")),
		mcp.WithNumber("limit", mcp.Description("Max results to return (default 50)")),
	)
}

func addToInboxTool() mcp.Tool {
	return mcp.NewTool("add_to_inbox",
		mcp.WithDescription("Add a work item to the user's inbox by its display ID (e.g. TF-141). Max 100 items per inbox."),
		mcp.WithString("display_id", mcp.Required(), mcp.Description("Work item display ID, e.g. TF-141")),
	)
}

func removeFromInboxTool() mcp.Tool {
	return mcp.NewTool("remove_from_inbox",
		mcp.WithDescription("Remove an item from the user's inbox"),
		mcp.WithString("inbox_item_id", mcp.Required(), mcp.Description("Inbox item UUID (from list_inbox)")),
	)
}

func reorderInboxTool() mcp.Tool {
	return mcp.NewTool("reorder_inbox",
		mcp.WithDescription("Change the position of an inbox item"),
		mcp.WithString("inbox_item_id", mcp.Required(), mcp.Description("Inbox item UUID (from list_inbox)")),
		mcp.WithNumber("position", mcp.Required(), mcp.Description("New position (0-based index)")),
	)
}

func inboxCountTool() mcp.Tool {
	return mcp.NewTool("inbox_count",
		mcp.WithDescription("Get the count of non-completed items in the user's inbox"),
	)
}

func clearCompletedInboxTool() mcp.Tool {
	return mcp.NewTool("clear_completed_inbox",
		mcp.WithDescription("Remove all completed work items from the user's inbox. Returns the number of items removed."),
	)
}

// --- Inbox handlers ---

func handleListInbox(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := ListInboxParams{
		Search:           request.GetString("search", ""),
		IncludeCompleted: request.GetString("include_completed", "") == "true",
		Limit:            request.GetInt("limit", 0),
	}

	list, err := client.ListInbox(params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list inbox: %v", err)), nil
	}

	if len(list.Items) == 0 {
		return mcp.NewToolResultText("Inbox is empty."), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Inbox (%d items)\n\n", list.Total)
	for _, item := range list.Items {
		fmt.Fprintf(&sb, "- **%s** %s [%s/%s] (%s) — %s/%s",
			item.DisplayID, item.Title, item.Status, item.Priority, item.Type,
			item.ProjectKey, item.ProjectName)
		if item.AssigneeDisplayName != "" {
			fmt.Fprintf(&sb, " → %s", item.AssigneeDisplayName)
		}
		if item.DueDate != nil {
			fmt.Fprintf(&sb, " due:%s", *item.DueDate)
		}
		fmt.Fprintf(&sb, " (inbox_item_id: %s)\n", item.ID)
	}
	if list.HasMore {
		sb.WriteString("\n_More items available._\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}

func handleAddToInbox(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	displayID, err := request.RequireString("display_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Look up the work item to get its UUID
	projectKey, itemNumber, err := parseDisplayID(displayID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	item, err := client.GetWorkItem(projectKey, itemNumber)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to find work item %s: %v", displayID, err)), nil
	}

	if err := client.AddToInbox(item.ID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add to inbox: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added %s (%s) to inbox.", displayID, item.Title)), nil
}

func handleRemoveFromInbox(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	inboxItemID, err := request.RequireString("inbox_item_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.RemoveFromInbox(inboxItemID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove from inbox: %v", err)), nil
	}

	return mcp.NewToolResultText("Removed item from inbox."), nil
}

func handleReorderInbox(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	inboxItemID, err := request.RequireString("inbox_item_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	position := request.GetInt("position", 0)

	if err := client.ReorderInboxItem(inboxItemID, position); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reorder inbox item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Moved inbox item to position %d.", position)), nil
}

func handleInboxCount(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	count, err := client.InboxCount()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get inbox count: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Inbox has %d non-completed items.", count)), nil
}

func handleClearCompletedInbox(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := getClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	removed, err := client.ClearCompletedInbox()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to clear completed inbox items: %v", err)), nil
	}

	if removed == 0 {
		return mcp.NewToolResultText("No completed items to clear."), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Removed %d completed items from inbox.", removed)), nil
}

// --- Formatting helpers ---

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func formatMilestone(m *Milestone) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", m.Name)
	fmt.Fprintf(&sb, "- **Status**: %s\n", m.Status)
	fmt.Fprintf(&sb, "- **Progress**: %d/%d closed (%d open)\n", m.ClosedCount, m.TotalCount, m.OpenCount)
	if m.DueDate != nil {
		fmt.Fprintf(&sb, "- **Due date**: %s\n", *m.DueDate)
	}
	if m.Description != nil && *m.Description != "" {
		fmt.Fprintf(&sb, "- **Description**: %s\n", *m.Description)
	}
	fmt.Fprintf(&sb, "- **ID**: %s\n", m.ID)
	fmt.Fprintf(&sb, "- **Created**: %s\n", m.CreatedAt)
	return sb.String()
}

func formatQueue(q *Queue) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", q.Name)
	fmt.Fprintf(&sb, "- **Type**: %s\n", q.QueueType)
	fmt.Fprintf(&sb, "- **Public**: %v\n", q.IsPublic)
	if q.DefaultPriority != "" {
		fmt.Fprintf(&sb, "- **Default priority**: %s\n", q.DefaultPriority)
	}
	if q.Description != nil && *q.Description != "" {
		fmt.Fprintf(&sb, "- **Description**: %s\n", *q.Description)
	}
	fmt.Fprintf(&sb, "- **ID**: %s\n", q.ID)
	fmt.Fprintf(&sb, "- **Created**: %s\n", q.CreatedAt)
	return sb.String()
}

// milestoneNameCache caches milestone ID → name mappings per project to avoid repeated API calls.
var milestoneNameCache = map[string]map[string]string{} // projectKey → milestoneID → name

// resolveMilestoneName returns the milestone name for a given ID, using the cache when possible.
func resolveMilestoneName(client *Client, projectKey, milestoneID string) string {
	if cache, ok := milestoneNameCache[projectKey]; ok {
		if name, ok := cache[milestoneID]; ok {
			return name
		}
	}

	// Fetch all milestones for the project and cache them
	milestones, err := client.ListMilestones(projectKey)
	if err != nil {
		return milestoneID // fallback to ID
	}

	cache := make(map[string]string, len(milestones))
	for _, m := range milestones {
		cache[m.ID] = m.Name
	}
	milestoneNameCache[projectKey] = cache

	if name, ok := cache[milestoneID]; ok {
		return name
	}
	return milestoneID
}

func formatWorkItemWithClient(item *WorkItem, client *Client) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s: %s\n\n", item.DisplayID, item.Title)
	fmt.Fprintf(&sb, "- **Type**: %s\n", item.Type)
	fmt.Fprintf(&sb, "- **Status**: %s\n", item.Status)
	fmt.Fprintf(&sb, "- **Priority**: %s\n", item.Priority)
	if item.AssigneeID != nil {
		fmt.Fprintf(&sb, "- **Assignee**: %s\n", *item.AssigneeID)
	} else {
		sb.WriteString("- **Assignee**: unassigned\n")
	}
	if item.MilestoneID != nil {
		name := *item.MilestoneID
		if client != nil {
			name = resolveMilestoneName(client, item.ProjectKey, *item.MilestoneID)
		}
		fmt.Fprintf(&sb, "- **Milestone**: %s\n", name)
	}
	if len(item.Labels) > 0 {
		fmt.Fprintf(&sb, "- **Labels**: %s\n", strings.Join(item.Labels, ", "))
	}
	if item.DueDate != nil {
		fmt.Fprintf(&sb, "- **Due date**: %s\n", *item.DueDate)
	}
	if item.ResolvedAt != nil {
		fmt.Fprintf(&sb, "- **Resolved**: %s\n", *item.ResolvedAt)
	}
	fmt.Fprintf(&sb, "- **Created**: %s\n", item.CreatedAt)
	fmt.Fprintf(&sb, "- **Updated**: %s\n", item.UpdatedAt)
	if item.Description != nil && *item.Description != "" {
		fmt.Fprintf(&sb, "\n### Description\n\n%s\n", *item.Description)
	}
	return sb.String()
}
