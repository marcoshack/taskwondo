package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"taskwondo",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// Auth tools
	s.AddTool(whoamiTool(), handleWhoami)
	s.AddTool(loginTool(), handleLogin)
	s.AddTool(logoutTool(), handleLogout)

	// Work item tools
	s.AddTool(getWorkItemTool(), handleGetWorkItem)
	s.AddTool(listWorkItemsTool(), handleListWorkItems)
	s.AddTool(createWorkItemTool(), handleCreateWorkItem)
	s.AddTool(updateWorkItemTool(), handleUpdateWorkItem)

	// Comment tools
	s.AddTool(addCommentTool(), handleAddComment)
	s.AddTool(listCommentsTool(), handleListComments)

	// Attachment tools
	s.AddTool(uploadAttachmentTool(), handleUploadAttachment)

	// Project tools
	s.AddTool(listProjectsTool(), handleListProjects)
	s.AddTool(getProjectTool(), handleGetProject)

	// Workflow tools
	s.AddTool(listStatusesTool(), handleListStatuses)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
