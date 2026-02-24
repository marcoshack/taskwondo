package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

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

	return mcp.NewToolResultText(formatWorkItem(item)), nil
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
		item.DisplayID, item.Title, formatWorkItem(item))), nil
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

	if len(updates) == 0 {
		return mcp.NewToolResultError("No fields to update. Provide at least one field to change."), nil
	}

	item, err := client.UpdateWorkItem(projectKey, itemNumber, updates)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to update work item: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Updated %s\n\n%s", item.DisplayID, formatWorkItem(item))), nil
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

// --- Formatting helpers ---

func formatWorkItem(item *WorkItem) string {
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
