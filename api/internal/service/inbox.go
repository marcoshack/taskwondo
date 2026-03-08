package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// InboxRepository defines persistence operations for inbox items.
type InboxRepository interface {
	Add(ctx context.Context, item *model.InboxItem) error
	Remove(ctx context.Context, userID, workItemID uuid.UUID) error
	RemoveByID(ctx context.Context, id, userID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, excludeCompleted bool, search string, projectKeys []string, cursor *uuid.UUID, limit int) (*model.InboxItemList, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountAllByUser(ctx context.Context, userID uuid.UUID) (int, error)
	UpdatePosition(ctx context.Context, id, userID uuid.UUID, position int) error
	MaxPosition(ctx context.Context, userID uuid.UUID) (int, error)
	RemoveCompleted(ctx context.Context, userID uuid.UUID) (int, error)
	Exists(ctx context.Context, userID, workItemID uuid.UUID) (bool, error)
	GetWorkItemProjectID(ctx context.Context, workItemID uuid.UUID) (uuid.UUID, error)
	RemoveByProjectID(ctx context.Context, projectID uuid.UUID) (int, error)
}

// InboxService handles inbox business logic.
type InboxService struct {
	inbox   InboxRepository
	members ProjectMemberRepository
}

// NewInboxService creates a new InboxService.
func NewInboxService(inbox InboxRepository, members ProjectMemberRepository) *InboxService {
	return &InboxService{
		inbox:   inbox,
		members: members,
	}
}

// Add adds a work item to the user's inbox.
func (s *InboxService) Add(ctx context.Context, info *model.AuthInfo, workItemID uuid.UUID) error {
	// Look up the work item's project for membership check
	projectID, err := s.inbox.GetWorkItemProjectID(ctx, workItemID)
	if err != nil {
		return fmt.Errorf("looking up work item: %w", err)
	}

	// Check the user is a member of the work item's project
	if err := s.requireMembership(ctx, info, projectID); err != nil {
		return err
	}

	// Check inbox limit
	count, err := s.inbox.CountAllByUser(ctx, info.UserID)
	if err != nil {
		return fmt.Errorf("counting inbox items: %w", err)
	}
	if count >= model.MaxInboxItems {
		return fmt.Errorf("%w: inbox is full (maximum %d items)", model.ErrValidation, model.MaxInboxItems)
	}

	// Check if already in inbox
	exists, err := s.inbox.Exists(ctx, info.UserID, workItemID)
	if err != nil {
		return fmt.Errorf("checking inbox existence: %w", err)
	}
	if exists {
		return fmt.Errorf("%w: item already in inbox", model.ErrAlreadyExists)
	}

	// Get next position
	maxPos, err := s.inbox.MaxPosition(ctx, info.UserID)
	if err != nil {
		return fmt.Errorf("getting max position: %w", err)
	}

	item := &model.InboxItem{
		ID:         uuid.New(),
		UserID:     info.UserID,
		WorkItemID: workItemID,
		Position:   maxPos + model.InboxPositionGap,
	}

	if err := s.inbox.Add(ctx, item); err != nil {
		return fmt.Errorf("adding inbox item: %w", err)
	}

	log.Ctx(ctx).Info().
		Stringer("work_item_id", workItemID).
		Stringer("user_id", info.UserID).
		Msg("added item to inbox")

	return nil
}

// Remove removes a work item from the user's inbox by inbox item ID.
func (s *InboxService) Remove(ctx context.Context, info *model.AuthInfo, inboxItemID uuid.UUID) error {
	if err := s.inbox.RemoveByID(ctx, inboxItemID, info.UserID); err != nil {
		return fmt.Errorf("removing inbox item: %w", err)
	}

	log.Ctx(ctx).Info().
		Stringer("inbox_item_id", inboxItemID).
		Stringer("user_id", info.UserID).
		Msg("removed item from inbox")

	return nil
}

// RemoveByWorkItem removes a work item from the user's inbox by work item ID.
func (s *InboxService) RemoveByWorkItem(ctx context.Context, info *model.AuthInfo, workItemID uuid.UUID) error {
	if err := s.inbox.Remove(ctx, info.UserID, workItemID); err != nil {
		return fmt.Errorf("removing inbox item: %w", err)
	}

	log.Ctx(ctx).Info().
		Stringer("work_item_id", workItemID).
		Stringer("user_id", info.UserID).
		Msg("removed item from inbox by work item ID")

	return nil
}

// List returns the user's inbox items.
func (s *InboxService) List(ctx context.Context, info *model.AuthInfo, excludeCompleted bool, search string, projectKeys []string, cursor *uuid.UUID, limit int) (*model.InboxItemList, error) {
	list, err := s.inbox.List(ctx, info.UserID, excludeCompleted, search, projectKeys, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("listing inbox items: %w", err)
	}
	return list, nil
}

// Count returns the number of active (non-completed) inbox items for the user.
func (s *InboxService) Count(ctx context.Context, info *model.AuthInfo) (int, error) {
	count, err := s.inbox.CountByUser(ctx, info.UserID)
	if err != nil {
		return 0, fmt.Errorf("counting inbox items: %w", err)
	}
	return count, nil
}

// Reorder updates the position of an inbox item.
func (s *InboxService) Reorder(ctx context.Context, info *model.AuthInfo, inboxItemID uuid.UUID, position int) error {
	if position < 0 {
		return fmt.Errorf("%w: position must be non-negative", model.ErrValidation)
	}

	if err := s.inbox.UpdatePosition(ctx, inboxItemID, info.UserID, position); err != nil {
		return fmt.Errorf("reordering inbox item: %w", err)
	}

	log.Ctx(ctx).Info().
		Stringer("inbox_item_id", inboxItemID).
		Int("position", position).
		Msg("reordered inbox item")

	return nil
}

// ClearCompleted removes all completed inbox items for the user.
func (s *InboxService) ClearCompleted(ctx context.Context, info *model.AuthInfo) (int, error) {
	count, err := s.inbox.RemoveCompleted(ctx, info.UserID)
	if err != nil {
		return 0, fmt.Errorf("clearing completed inbox items: %w", err)
	}

	log.Ctx(ctx).Info().
		Stringer("user_id", info.UserID).
		Int("removed", count).
		Msg("cleared completed inbox items")

	return count, nil
}

func (s *InboxService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
	if info.GlobalRole == model.RoleAdmin {
		return nil
	}
	_, err := s.members.GetByProjectAndUser(ctx, projectID, info.UserID)
	if err != nil {
		if err == model.ErrNotFound {
			return model.ErrNotFound
		}
		return fmt.Errorf("checking membership: %w", err)
	}
	return nil
}
