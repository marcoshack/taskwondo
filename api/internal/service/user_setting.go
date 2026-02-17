package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/marcoshack/trackforge/internal/model"
)

// UserSettingRepository defines persistence operations for user settings.
type UserSettingRepository interface {
	Upsert(ctx context.Context, s *model.UserSetting) error
	Get(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) (*model.UserSetting, error)
	ListByProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) ([]model.UserSetting, error)
	Delete(ctx context.Context, userID uuid.UUID, projectID *uuid.UUID, key string) error
}

// UserSettingService handles user setting business logic.
type UserSettingService struct {
	settings UserSettingRepository
	projects ProjectRepository
	members  ProjectMemberRepository
}

// NewUserSettingService creates a new UserSettingService.
func NewUserSettingService(settings UserSettingRepository, projects ProjectRepository, members ProjectMemberRepository) *UserSettingService {
	return &UserSettingService{
		settings: settings,
		projects: projects,
		members:  members,
	}
}

// Set creates or updates a user setting for the given project.
func (s *UserSettingService) Set(ctx context.Context, info *model.AuthInfo, projectKey string, key string, value json.RawMessage) (*model.UserSetting, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	setting := &model.UserSetting{
		UserID:    info.UserID,
		ProjectID: &project.ID,
		Key:       key,
		Value:     value,
	}

	if err := s.settings.Upsert(ctx, setting); err != nil {
		return nil, fmt.Errorf("saving user setting: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("project_key", projectKey).
		Str("key", key).
		Msg("user setting saved")

	return s.settings.Get(ctx, info.UserID, &project.ID, key)
}

// Get returns a single user setting.
func (s *UserSettingService) Get(ctx context.Context, info *model.AuthInfo, projectKey string, key string) (*model.UserSetting, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.settings.Get(ctx, info.UserID, &project.ID, key)
}

// List returns all settings for the user in the given project.
func (s *UserSettingService) List(ctx context.Context, info *model.AuthInfo, projectKey string) ([]model.UserSetting, error) {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return nil, err
	}

	return s.settings.ListByProject(ctx, info.UserID, project.ID)
}

// Delete removes a user setting.
func (s *UserSettingService) Delete(ctx context.Context, info *model.AuthInfo, projectKey string, key string) error {
	project, err := s.projects.GetByKey(ctx, projectKey)
	if err != nil {
		return err
	}

	if err := s.requireMembership(ctx, info, project.ID); err != nil {
		return err
	}

	return s.settings.Delete(ctx, info.UserID, &project.ID, key)
}

func (s *UserSettingService) requireMembership(ctx context.Context, info *model.AuthInfo, projectID uuid.UUID) error {
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
