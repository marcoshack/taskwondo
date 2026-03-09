package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/marcoshack/taskwondo/internal/model"
)

// AdminUserRepository defines user persistence operations needed by the admin service.
type AdminUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	ListAll(ctx context.Context) ([]model.User, error)
	UpdateGlobalRole(ctx context.Context, id uuid.UUID, role string) error
	UpdateIsActive(ctx context.Context, id uuid.UUID, isActive bool) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, forceChange bool) error
	UpdateMaxProjects(ctx context.Context, id uuid.UUID, maxProjects *int) error
	UpdateMaxNamespaces(ctx context.Context, id uuid.UUID, maxNamespaces *int) error
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
	CountByRole(ctx context.Context, projectID uuid.UUID, role string) (int, error)
	UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role string) error
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

// UpdateUser updates a user's global role, active status, project limit, and/or namespace limit.
func (s *AdminService) UpdateUser(ctx context.Context, callerID, targetID uuid.UUID, role *string, isActive *bool, maxProjects *int, maxNamespaces *int) (*model.User, error) {
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

	if maxProjects != nil {
		if *maxProjects < -1 {
			return nil, fmt.Errorf("%w: max_projects must be -1 (default), 0 (unlimited), or a positive number", model.ErrValidation)
		}
		var dbVal *int
		if *maxProjects >= 0 {
			dbVal = maxProjects
		}
		// *maxProjects == -1 means clear (reset to global default), dbVal stays nil
		if err := s.users.UpdateMaxProjects(ctx, targetID, dbVal); err != nil {
			return nil, err
		}
	}

	if maxNamespaces != nil {
		if *maxNamespaces < -1 {
			return nil, fmt.Errorf("%w: max_namespaces must be -1 (default), 0 (unlimited), or a positive number", model.ErrValidation)
		}
		var dbVal *int
		if *maxNamespaces >= 0 {
			dbVal = maxNamespaces
		}
		// *maxNamespaces == -1 means clear (reset to global default), dbVal stays nil
		if err := s.users.UpdateMaxNamespaces(ctx, targetID, dbVal); err != nil {
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

// UpdateUserProjectRole updates a user's role in a project.
func (s *AdminService) UpdateUserProjectRole(ctx context.Context, userID, projectID uuid.UUID, role string) error {
	switch role {
	case model.ProjectRoleOwner, model.ProjectRoleAdmin, model.ProjectRoleMember, model.ProjectRoleViewer:
	default:
		return fmt.Errorf("%w: invalid project role %q", model.ErrValidation, role)
	}

	// Check membership exists
	target, err := s.members.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Protect last owner
	if target.Role == model.ProjectRoleOwner && role != model.ProjectRoleOwner {
		count, err := s.members.CountByRole(ctx, projectID, model.ProjectRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot remove the last owner: %w", model.ErrConflict)
		}
	}

	return s.members.UpdateRole(ctx, projectID, userID, role)
}

// RemoveUserFromProject removes a user from a project.
func (s *AdminService) RemoveUserFromProject(ctx context.Context, userID, projectID uuid.UUID) error {
	// Check membership exists and protect last owner
	target, err := s.members.GetByProjectAndUser(ctx, projectID, userID)
	if err != nil {
		return err
	}

	if target.Role == model.ProjectRoleOwner {
		count, err := s.members.CountByRole(ctx, projectID, model.ProjectRoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return fmt.Errorf("cannot remove the last owner: %w", model.ErrConflict)
		}
	}

	return s.members.Remove(ctx, projectID, userID)
}

// CreateUser creates a new user with a temporary password.
// Returns the created user and the plaintext temporary password (shown once).
func (s *AdminService) CreateUser(ctx context.Context, email, displayName string) (*model.User, string, error) {
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, "", fmt.Errorf("%w: email is required", model.ErrValidation)
	}
	if displayName == "" {
		return nil, "", fmt.Errorf("%w: display name is required", model.ErrValidation)
	}

	// Check email uniqueness
	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, "", model.ErrAlreadyExists
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, "", fmt.Errorf("checking email: %w", err)
	}

	password, err := generateTemporaryPassword()
	if err != nil {
		return nil, "", fmt.Errorf("generating password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		ID:                  uuid.New(),
		Email:               email,
		DisplayName:         displayName,
		PasswordHash:        string(hash),
		GlobalRole:          model.RoleUser,
		IsActive:            true,
		ForcePasswordChange: true,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("creating user: %w", err)
	}

	return user, password, nil
}

// ResetUserPassword generates a new temporary password for a user.
// Returns the plaintext temporary password (shown once).
func (s *AdminService) ResetUserPassword(ctx context.Context, userID uuid.UUID) (string, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return "", err
	}

	password, err := generateTemporaryPassword()
	if err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	if err := s.users.UpdatePasswordHash(ctx, userID, string(hash), true); err != nil {
		return "", fmt.Errorf("updating password: %w", err)
	}

	return password, nil
}

func generateTemporaryPassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}
