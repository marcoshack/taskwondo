package service

import (
	"context"
	"testing"

	"github.com/marcoshack/taskwondo/internal/model"
)

// inlineTestItem creates a project, membership and a work item carrying the
// given multi-line description, returning everything a CreateInlineComment
// call needs.
func inlineTestItem(t *testing.T, desc string) (*testWorkItemSetup, *model.AuthInfo, string, int) {
	t.Helper()
	setup := newTestWorkItemSetup()
	info := userAuthInfo()
	project := setupProjectWithMember(t, setup.projectRepo, setup.memberRepo, info, model.ProjectRoleOwner)
	in := validCreateInput()
	in.Description = &desc
	item, err := setup.svc.Create(context.Background(), info, project.Key, in)
	if err != nil {
		t.Fatalf("creating work item: %v", err)
	}
	return setup, info, project.Key, item.ItemNumber
}

func TestCreateInlineComment_RootStoresColumns(t *testing.T) {
	desc := "Line one here\nLine two here\nLine three"
	setup, info, key, num := inlineTestItem(t, desc)

	c, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"a comment", model.VisibilityInternal,
		InlineAnchorInput{
			StartLine: 1, StartCol: 6,
			EndLine: 2, EndCol: 4,
			Snippet: "Line one here\nLine two here",
		}, nil)
	if err != nil {
		t.Fatalf("CreateInlineComment: %v", err)
	}
	if c.Anchor == nil {
		t.Fatal("expected anchor on root inline comment")
	}
	if c.Anchor.StartCol != 6 || c.Anchor.EndCol != 4 {
		t.Errorf("columns not stored: got start=%d end=%d", c.Anchor.StartCol, c.Anchor.EndCol)
	}
	if c.Anchor.StartLine != 1 || c.Anchor.EndLine != 2 {
		t.Errorf("lines: got start=%d end=%d", c.Anchor.StartLine, c.Anchor.EndLine)
	}
	if c.Anchor.Status != model.AnchorStatusActive {
		t.Errorf("expected active anchor, got %q", c.Anchor.Status)
	}
	if c.ParentCommentID != nil {
		t.Error("root comment should have no parent")
	}
}

func TestCreateInlineComment_ReplyInheritsAnchor(t *testing.T) {
	desc := "Line one here\nLine two here\nLine three"
	setup, info, key, num := inlineTestItem(t, desc)

	root, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"root", model.VisibilityInternal,
		InlineAnchorInput{StartLine: 1, StartCol: 6, EndLine: 2, EndCol: 4, Snippet: "Line one here\nLine two here"}, nil)
	if err != nil {
		t.Fatalf("creating root: %v", err)
	}

	reply, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"a reply", model.VisibilityInternal, InlineAnchorInput{}, &root.ID)
	if err != nil {
		t.Fatalf("creating reply: %v", err)
	}
	if reply.ParentCommentID == nil || *reply.ParentCommentID != root.ID {
		t.Fatalf("reply parent = %v, want %v", reply.ParentCommentID, root.ID)
	}
	if reply.Anchor == nil {
		t.Fatal("reply should inherit an anchor")
	}
	if reply.Anchor.StartLine != root.Anchor.StartLine ||
		reply.Anchor.StartCol != root.Anchor.StartCol ||
		reply.Anchor.EndLine != root.Anchor.EndLine ||
		reply.Anchor.EndCol != root.Anchor.EndCol ||
		reply.Anchor.Snippet != root.Anchor.Snippet ||
		reply.Anchor.RevisionID != root.Anchor.RevisionID {
		t.Errorf("reply anchor %+v does not match root anchor %+v", reply.Anchor, root.Anchor)
	}
}

func TestCreateInlineComment_ReplyToReplyReroots(t *testing.T) {
	desc := "Line one here\nLine two here\nLine three"
	setup, info, key, num := inlineTestItem(t, desc)

	root, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"root", model.VisibilityInternal,
		InlineAnchorInput{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 5, Snippet: "Line one here"}, nil)
	if err != nil {
		t.Fatalf("creating root: %v", err)
	}
	reply1, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"reply1", model.VisibilityInternal, InlineAnchorInput{}, &root.ID)
	if err != nil {
		t.Fatalf("creating reply1: %v", err)
	}
	// Reply to the reply — must flatten back onto the root.
	reply2, err := setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"reply2", model.VisibilityInternal, InlineAnchorInput{}, &reply1.ID)
	if err != nil {
		t.Fatalf("creating reply2: %v", err)
	}
	if reply2.ParentCommentID == nil || *reply2.ParentCommentID != root.ID {
		t.Errorf("reply-to-reply parent = %v, want root %v", reply2.ParentCommentID, root.ID)
	}
}

func TestCreateInlineComment_ReplyRejectsNonInlineParent(t *testing.T) {
	desc := "Line one here\nLine two here"
	setup, info, key, num := inlineTestItem(t, desc)

	plain, err := setup.svc.CreateComment(context.Background(), info, key, num, CreateCommentInput{
		Body: "regular comment", Visibility: model.VisibilityInternal,
	})
	if err != nil {
		t.Fatalf("creating plain comment: %v", err)
	}
	_, err = setup.svc.CreateInlineComment(context.Background(), info, key, num,
		"reply", model.VisibilityInternal, InlineAnchorInput{}, &plain.ID)
	if err == nil {
		t.Fatal("expected error replying to a non-inline comment")
	}
}
