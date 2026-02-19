package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// AdminUserRepository defines user persistence operations needed by the admin service.
type AdminUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	ListAll(ctx context.Context) ([]model.User, error)
	UpdateGlobalRole(ctx context.Context, id uuid.UUID, role string) error
	UpdateIsActive(ctx context.Context, id uuid.UUID, isActive bool) error
	CountByRole(ctx context.Context, role string) (int, error)
}

// AdminProjectRepository defines project persistence operations needed by the admin service.
type AdminProjectRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
}

// AdminMemberRepository defines project member persistence operations needed by the admin service.
type AdminMemberRepository interface {
	Add(ctx context.Context, member *model.ProjectMember) error
	GetByProjectAndUser(ctx context.Context, projectID, userID uuid.UUID) (*model.ProjectMember, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error)
	Remove(ctx context.Context, projectID, userID uuid.UUID) error
}

// AdminService handles admin-only business logic.
type AdminService struct {
	users    AdminUserRepository
	projects AdminProjectRepository
	members  AdminMemberRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(users AdminUserRepository, projects AdminProjectRepository, members AdminMemberRepository) *AdminService {
	return &AdminService{users: users, projects: projects, members: members}
}

// ListUsers returns all users in the system.
func (s *AdminService) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.users.ListAll(ctx)
}

// UpdateUser updates a user's global role and/or active status.
func (s *AdminService) UpdateUser(ctx context.Context, callerID, targetID uuid.UUID, role *string, isActive *bool) (*model.User, error) {
	// Prevent self-modification for role and active status
	if callerID == targetID {
		if role != nil {
			return nil, fmt.Errorf("%w: cannot change your own role", model.ErrValidation)
		}
		if isActive != nil && !*isActive {
			return nil, fmt.Errorf("%w: cannot disable your own account", model.ErrValidation)
		}
	}

	if role != nil {
		if *role != model.RoleAdmin && *role != model.RoleUser {
			return nil, fmt.Errorf("%w: invalid role %q", model.ErrValidation, *role)
		}

		// If demoting from admin, ensure at least one admin remains
		target, err := s.users.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if target.GlobalRole == model.RoleAdmin && *role != model.RoleAdmin {
			count, err := s.users.CountByRole(ctx, model.RoleAdmin)
			if err != nil {
				return nil, fmt.Errorf("counting admins: %w", err)
			}
			if count <= 1 {
				return nil, fmt.Errorf("%w: cannot remove the last admin", model.ErrValidation)
			}
		}

		if err := s.users.UpdateGlobalRole(ctx, targetID, *role); err != nil {
			return nil, err
		}
	}

	if isActive != nil {
		if err := s.users.UpdateIsActive(ctx, targetID, *isActive); err != nil {
			return nil, err
		}
	}

	return s.users.GetByID(ctx, targetID)
}

// ListUserProjects returns all project memberships for a user.
func (s *AdminService) ListUserProjects(ctx context.Context, userID uuid.UUID) ([]model.ProjectMemberWithProject, error) {
	return s.members.ListByUser(ctx, userID)
}

// AddUserToProject adds a user to a project with the given role.
func (s *AdminService) AddUserToProject(ctx context.Context, userID, projectID uuid.UUID, role string) error {
	// Validate role
	switch role {
	case model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember, model.ProjectRoleViewer:
	default:
		return fmt.Errorf("%w: invalid project role %q", model.ErrValidation, role)
	}

	// Check user exists
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return err
	}

	// Check project exists
	if _, err := s.projects.GetByID(ctx, projectID); err != nil {
		return err
	}

	// Check not already a member
	if _, err := s.members.GetByProjectAndUser(ctx, projectID, userID); err == nil {
		return model.ErrAlreadyExists
	} else if !errors.Is(err, model.ErrNotFound) {
		return err
	}

	member := &model.ProjectMember{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
	}

	return s.members.Add(ctx, member)
}

// RemoveUserFromProject removes a user from a project.
func (s *AdminService) RemoveUserFromProject(ctx context.Context, userID, projectID uuid.UUID) error {
	return s.members.Remove(ctx, projectID, userID)
}
