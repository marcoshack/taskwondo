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

	// Activity / relation tools
	s.AddTool(listEventsTool(), handleListEvents)
	s.AddTool(createRelationTool(), handleCreateRelation)
	s.AddTool(listRelationsTool(), handleListRelations)

	// Attachment tools
	s.AddTool(listAttachmentsTool(), handleListAttachments)
	s.AddTool(uploadAttachmentTool(), handleUploadAttachment)

	// Project tools
	s.AddTool(listProjectsTool(), handleListProjects)
	s.AddTool(getProjectTool(), handleGetProject)

	// Time tracking tools
	s.AddTool(logTimeTool(), handleLogTime)
	s.AddTool(listTimeEntriesTool(), handleListTimeEntries)
	s.AddTool(updateTimeEntryTool(), handleUpdateTimeEntry)
	s.AddTool(deleteTimeEntryTool(), handleDeleteTimeEntry)

	// Workflow tools
	s.AddTool(listStatusesTool(), handleListStatuses)

	// Extended work item tools
	s.AddTool(deleteWorkItemTool(), handleDeleteWorkItem)
	s.AddTool(updateCommentTool(), handleUpdateComment)
	s.AddTool(deleteCommentTool(), handleDeleteComment)
	s.AddTool(deleteRelationTool(), handleDeleteRelation)
	s.AddTool(deleteAttachmentTool(), handleDeleteAttachment)

	// Project management tools
	s.AddTool(createProjectTool(), handleCreateProject)
	s.AddTool(updateProjectTool(), handleUpdateProject)
	s.AddTool(deleteProjectTool(), handleDeleteProject)
	s.AddTool(listMembersTool(), handleListMembers)
	s.AddTool(addMemberTool(), handleAddMember)
	s.AddTool(updateMemberRoleTool(), handleUpdateMemberRole)
	s.AddTool(removeMemberTool(), handleRemoveMember)

	// User tools
	s.AddTool(searchUsersTool(), handleSearchUsers)

	// Milestone tools
	s.AddTool(listMilestonesTool(), handleListMilestones)
	s.AddTool(createMilestoneTool(), handleCreateMilestone)
	s.AddTool(getMilestoneTool(), handleGetMilestone)
	s.AddTool(updateMilestoneTool(), handleUpdateMilestone)
	s.AddTool(deleteMilestoneTool(), handleDeleteMilestone)

	// Queue tools
	s.AddTool(listQueuesTool(), handleListQueues)
	s.AddTool(createQueueTool(), handleCreateQueue)
	s.AddTool(getQueueTool(), handleGetQueue)
	s.AddTool(updateQueueTool(), handleUpdateQueue)
	s.AddTool(deleteQueueTool(), handleDeleteQueue)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
