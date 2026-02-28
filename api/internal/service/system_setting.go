package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/marcoshack/taskwondo/internal/model"
)

// SystemSettingRepositoryInterface defines persistence operations for system settings.
type SystemSettingRepositoryInterface interface {
	Upsert(ctx context.Context, s *model.SystemSetting) error
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
	List(ctx context.Context) ([]model.SystemSetting, error)
	Delete(ctx context.Context, key string) error
}

// SystemSettingService handles system setting business logic.
type SystemSettingService struct {
	settings SystemSettingRepositoryInterface
}

// NewSystemSettingService creates a new SystemSettingService.
func NewSystemSettingService(settings SystemSettingRepositoryInterface) *SystemSettingService {
	return &SystemSettingService{settings: settings}
}

// Set creates or updates a system setting. Requires admin role.
func (s *SystemSettingService) Set(ctx context.Context, info *model.AuthInfo, key string, value json.RawMessage) (*model.SystemSetting, error) {
	if err := requireAdmin(info); err != nil {
		return nil, err
	}

	setting := &model.SystemSetting{
		Key:   key,
		Value: value,
	}

	if err := s.settings.Upsert(ctx, setting); err != nil {
		return nil, fmt.Errorf("saving system setting: %w", err)
	}

	log.Ctx(ctx).Info().
		Str("key", key).
		Msg("system setting saved")

	return s.settings.Get(ctx, key)
}

// Get returns a single system setting.
func (s *SystemSettingService) Get(ctx context.Context, info *model.AuthInfo, key string) (*model.SystemSetting, error) {
	if err := requireAdmin(info); err != nil {
		return nil, err
	}
	return s.settings.Get(ctx, key)
}

// List returns all system settings. Requires admin role.
func (s *SystemSettingService) List(ctx context.Context, info *model.AuthInfo) ([]model.SystemSetting, error) {
	if err := requireAdmin(info); err != nil {
		return nil, err
	}
	return s.settings.List(ctx)
}

// Delete removes a system setting. Requires admin role.
func (s *SystemSettingService) Delete(ctx context.Context, info *model.AuthInfo, key string) error {
	if err := requireAdmin(info); err != nil {
		return err
	}
	return s.settings.Delete(ctx, key)
}

// GetPublic returns a curated set of system settings accessible without authentication.
func (s *SystemSettingService) GetPublic(ctx context.Context) (map[string]json.RawMessage, error) {
	publicKeys := []string{
		"brand_name",
		model.SettingMaxProjectsPerUser,
		model.SettingAuthEmailLoginEnabled,
		model.SettingAuthEmailRegistrationEnabled,
		model.SettingAuthDiscordEnabled,
		model.SettingAuthGoogleEnabled,
		model.SettingAuthGitHubEnabled,
		model.SettingFeatureStatsTimeline,
	}
	result := make(map[string]json.RawMessage)

	for _, key := range publicKeys {
		setting, err := s.settings.Get(ctx, key)
		if err != nil {
			if err == model.ErrNotFound {
				continue
			}
			return nil, fmt.Errorf("getting public setting %s: %w", key, err)
		}
		result[key] = setting.Value
	}

	return result, nil
}

func requireAdmin(info *model.AuthInfo) error {
	if info == nil || info.GlobalRole != model.RoleAdmin {
		return model.ErrForbidden
	}
	return nil
}
