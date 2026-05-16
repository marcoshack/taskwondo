package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// InlineAnchorInput describes the optional anchor sent when creating a
// comment. start_line / end_line are 1-based and inclusive. start_col /
// end_col are 1-based character offsets within their line (end_col is
// exclusive); 0 means a whole-block anchor. Snippet is the whole source
// lines [start_line, end_line] — the unit the re-anchor pass matches on.
type InlineAnchorInput struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	Snippet   string
}

// recordDescriptionRevision writes a new revision row for the work item's
// description and runs the re-anchor pass over its active inline comments.
// Identical (normalized) content is a no-op — no revision row is written.
func (s *WorkItemService) recordDescriptionRevision(ctx context.Context, workItemID uuid.UUID, info *model.AuthInfo, newContent string) error {
	if s.revisions == nil {
		return nil
	}

	newHash := HashContent(newContent)

	latest, err := s.revisions.GetLatest(ctx, workItemID)
	if err == nil && latest.ContentHash == newHash {
		// No-op: normalized content matches the latest revision.
		return nil
	}
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("fetching latest revision: %w", err)
	}

	authorID := info.UserID
	rev := &model.DescriptionRevision{
		ID:          uuid.Must(uuid.NewV7()),
		WorkItemID:  workItemID,
		Content:     NormalizeContent(newContent),
		ContentHash: newHash,
		AuthorID:    &authorID,
	}
	if err := s.revisions.Create(ctx, rev); err != nil {
		return fmt.Errorf("creating revision: %w", err)
	}

	s.reanchorComments(ctx, workItemID, info, rev)
	return nil
}

// reanchorComments scans every active inline comment on the work item and
// either re-anchors it against the new revision or marks it outdated. The
// pass is best-effort: if a single comment's update fails, it logs and
// continues with the next.
func (s *WorkItemService) reanchorComments(ctx context.Context, workItemID uuid.UUID, info *model.AuthInfo, newRev *model.DescriptionRevision) {
	if s.comments == nil {
		return
	}
	comments, err := s.comments.ListInlineByWorkItem(ctx, workItemID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("listing inline comments for re-anchor failed")
		return
	}

	for i := range comments {
		c := &comments[i]
		if c.Anchor == nil || c.Anchor.Status != model.AnchorStatusActive {
			continue
		}

		match := FindAnchor(newRev.Content, c.Anchor.Snippet)
		if match.Found {
			// Columns are kept as-is: FindAnchor matches whole lines, so the
			// selection's character offsets within the first/last line remain
			// valid. Best-effort for fuzzy matches where a line shifted.
			newAnchor := &model.CommentAnchor{
				RevisionID:     newRev.ID,
				RevisionNumber: newRev.RevisionNumber,
				StartLine:      match.StartLine,
				StartCol:       c.Anchor.StartCol,
				EndLine:        match.EndLine,
				EndCol:         c.Anchor.EndCol,
				Snippet:        match.Snippet,
				SnippetHash:    HashSnippet(match.Snippet),
				Status:         model.AnchorStatusActive,
			}
			if err := s.comments.UpdateAnchor(ctx, c.ID, newAnchor); err != nil {
				log.Ctx(ctx).Warn().Err(err).Str("comment_id", c.ID.String()).
					Msg("re-anchoring comment failed")
			}
			continue
		}

		// Snippet vanished — mark outdated and freeze the anchor at the
		// original revision so the diff modal still works.
		outdated := *c.Anchor
		outdated.Status = model.AnchorStatusOutdated
		if err := s.comments.UpdateAnchor(ctx, c.ID, &outdated); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("comment_id", c.ID.String()).
				Msg("marking comment outdated failed")
			continue
		}
		s.recordEventWithMetadata(ctx, workItemID, info, "inline_comment_outdated", c.Visibility, map[string]any{
			"comment_id": c.ID.String(),
		})
	}
}

// EnsureInitialRevision creates a revision row containing the work item's
// current description if no revisions exist for the item yet. This handles
// items created before TF-350 / outside the back-fill window.
func (s *WorkItemService) EnsureInitialRevision(ctx context.Context, item *model.WorkItem) (*model.DescriptionRevision, error) {
	if s.revisions == nil {
		return nil, fmt.Errorf("revisions not configured: %w", model.ErrValidation)
	}
	latest, err := s.revisions.GetLatest(ctx, item.ID)
	if err == nil {
		return latest, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	desc := ""
	if item.Description != nil {
		desc = *item.Description
	}
	rev := &model.DescriptionRevision{
		ID:          uuid.Must(uuid.NewV7()),
		WorkItemID:  item.ID,
		Content:     NormalizeContent(desc),
		ContentHash: HashContent(desc),
		AuthorID:    &item.ReporterID,
	}
	if err := s.revisions.Create(ctx, rev); err != nil {
		return nil, fmt.Errorf("creating initial revision: %w", err)
	}
	return rev, nil
}

// ListDescriptionRevisions returns the revision history for a work item.
func (s *WorkItemService) ListDescriptionRevisions(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int) ([]model.DescriptionRevision, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}
	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}
	if s.revisions == nil {
		return nil, nil
	}
	revs, err := s.revisions.ListByWorkItem(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	if len(revs) == 0 {
		// Lazily create an initial revision so subsequent reads are stable.
		if _, err := s.EnsureInitialRevision(ctx, item); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("ensuring initial revision failed")
			return nil, nil
		}
		return s.revisions.ListByWorkItem(ctx, item.ID)
	}
	return revs, nil
}

// GetDescriptionRevision returns a specific revision by ID, scoped to the work item.
func (s *WorkItemService) GetDescriptionRevision(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, revID uuid.UUID) (*model.DescriptionRevision, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}
	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}
	if s.revisions == nil {
		return nil, model.ErrNotFound
	}
	rev, err := s.revisions.GetByID(ctx, revID)
	if err != nil {
		return nil, err
	}
	if rev.WorkItemID != item.ID {
		return nil, model.ErrNotFound
	}
	return rev, nil
}

// CreateInlineComment creates a comment anchored to a region of the work
// item's current description. The anchor is resolved against the latest
// revision (creating one lazily if none exist).
//
// When parentCommentID is non-nil the new comment is a threaded reply: it
// inherits the anchor of the thread's root comment verbatim and skips the
// snippet-resolution pass, so a reply always stays anchored alongside the
// comment it answers.
func (s *WorkItemService) CreateInlineComment(ctx context.Context, info *model.AuthInfo, projectKey string, itemNumber int, body string, visibility string, anchor InlineAnchorInput, parentCommentID *uuid.UUID) (*model.Comment, error) {
	if s.revisions == nil {
		return nil, fmt.Errorf("inline comments are not configured: %w", model.ErrValidation)
	}

	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireRole(ctx, info, project.ID,
		model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember); err != nil {
		return nil, err
	}

	item, err := s.items.GetByProjectAndNumber(ctx, project.ID, itemNumber)
	if err != nil {
		return nil, err
	}

	if body == "" {
		return nil, fmt.Errorf("body is required: %w", model.ErrValidation)
	}
	if visibility == "" {
		visibility = model.VisibilityInternal
	}
	if !isValidVisibility(visibility) {
		return nil, fmt.Errorf("invalid comment visibility %q: %w", visibility, model.ErrValidation)
	}

	rev, err := s.EnsureInitialRevision(ctx, item)
	if err != nil {
		return nil, err
	}

	var resolvedAnchor *model.CommentAnchor
	var rootID *uuid.UUID

	if parentCommentID != nil {
		// Threaded reply: inherit the root comment's anchor.
		parent, err := s.comments.GetByID(ctx, *parentCommentID)
		if err != nil {
			return nil, fmt.Errorf("loading parent comment: %w", err)
		}
		if parent.WorkItemID != item.ID || parent.Anchor == nil {
			return nil, fmt.Errorf("parent comment is not an inline comment on this item: %w", model.ErrValidation)
		}
		// Flatten: replying to a reply re-roots onto the thread's root.
		root := parent
		if parent.ParentCommentID != nil {
			root, err = s.comments.GetByID(ctx, *parent.ParentCommentID)
			if err != nil {
				return nil, fmt.Errorf("loading thread root: %w", err)
			}
		}
		if root.Anchor == nil {
			return nil, fmt.Errorf("thread root has no anchor: %w", model.ErrValidation)
		}
		rootID = &root.ID
		inherited := *root.Anchor
		resolvedAnchor = &inherited
	} else {
		// Root inline comment: resolve the anchor against the description.
		if anchor.StartLine < 1 || anchor.EndLine < anchor.StartLine {
			return nil, fmt.Errorf("invalid anchor line range: %w", model.ErrValidation)
		}
		if anchor.Snippet == "" {
			return nil, fmt.Errorf("anchor snippet is required: %w", model.ErrValidation)
		}

		// Verify the snippet still maps somewhere in the current description;
		// if the client raced an edit, the anchor may already be invalid.
		desc := ""
		if item.Description != nil {
			desc = *item.Description
		}
		match := FindAnchor(desc, anchor.Snippet)
		status := model.AnchorStatusActive
		if !match.Found {
			status = model.AnchorStatusOutdated
		} else {
			anchor.StartLine = match.StartLine
			anchor.EndLine = match.EndLine
			anchor.Snippet = match.Snippet
		}
		resolvedAnchor = &model.CommentAnchor{
			RevisionID:     rev.ID,
			RevisionNumber: rev.RevisionNumber,
			StartLine:      anchor.StartLine,
			StartCol:       anchor.StartCol,
			EndLine:        anchor.EndLine,
			EndCol:         anchor.EndCol,
			Snippet:        anchor.Snippet,
			SnippetHash:    HashSnippet(anchor.Snippet),
			Status:         status,
		}
	}

	comment := &model.Comment{
		ID:              uuid.Must(uuid.NewV7()),
		WorkItemID:      item.ID,
		AuthorID:        &info.UserID,
		Body:            body,
		Visibility:      visibility,
		Anchor:          resolvedAnchor,
		ParentCommentID: rootID,
	}

	if err := s.comments.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("creating inline comment: %w", err)
	}

	_ = s.items.TouchUpdatedAt(ctx, item.ID)

	eventName := "inline_comment_added"
	if rootID != nil {
		eventName = "inline_comment_replied"
	}
	meta := map[string]any{
		"comment_id":      comment.ID.String(),
		"revision_id":     resolvedAnchor.RevisionID.String(),
		"revision_number": resolvedAnchor.RevisionNumber,
		"start_line":      resolvedAnchor.StartLine,
		"end_line":        resolvedAnchor.EndLine,
		"preview":         body,
	}
	if rootID != nil {
		meta["parent_comment_id"] = rootID.String()
	}
	s.recordEventWithMetadata(ctx, item.ID, info, eventName, visibility, meta)

	preview := body
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	s.publishWatcherNotification(ctx, projectKey, project.ID, item, info.UserID, "comment_added", "", "", "", preview)
	s.publishCommentOnAssigned(ctx, projectKey, project.ID, item, info.UserID, preview)
	publishEmbedIndex(ctx, s.publisher, s.embedCache, model.EntityTypeComment, comment.ID, &project.ID)

	return s.comments.GetByID(ctx, comment.ID)
}
