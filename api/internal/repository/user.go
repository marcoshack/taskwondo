package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/taskwondo/internal/model"
)

// UserRepository handles user persistence.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByEmail returns a user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, password_hash, global_role, avatar_url,
		        is_active, force_password_change, max_projects, max_namespaces, last_login_at, created_at, updated_at
		 FROM users WHERE email = $1`, email)

	return scanUser(row)
}

// GetByID returns a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, password_hash, global_role, avatar_url,
		        is_active, force_password_change, max_projects, max_namespaces, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`, id)

	return scanUser(row)
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, display_name, password_hash, global_role, avatar_url, is_active, force_password_change)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.Email, user.DisplayName, sql.NullString{String: user.PasswordHash, Valid: user.PasswordHash != ""},
		user.GlobalRole, user.AvatarURL, user.IsActive, user.ForcePasswordChange)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// UpdateLastLogin sets the last_login_at timestamp to now.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = now(), updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("updating last login: %w", err)
	}
	return nil
}

// UpdateDisplayName sets the display_name for a user.
func (r *UserRepository) UpdateDisplayName(ctx context.Context, id uuid.UUID, displayName string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = $1, updated_at = now() WHERE id = $2`,
		displayName, id)
	if err != nil {
		return fmt.Errorf("updating display name: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// UpdateAvatarURL sets the avatar_url for a user.
func (r *UserRepository) UpdateAvatarURL(ctx context.Context, id uuid.UUID, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET avatar_url = $1, updated_at = now() WHERE id = $2`,
		sql.NullString{String: avatarURL, Valid: avatarURL != ""}, id)
	if err != nil {
		return fmt.Errorf("updating avatar url: %w", err)
	}
	return nil
}

// Search returns active users visible to the caller (co-project members).
// When query is non-empty, results are filtered by email or display_name (ILIKE).
// Results are limited to 20.
func (r *UserRepository) Search(ctx context.Context, callerID uuid.UUID, query string) ([]model.User, error) {
	q := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT u.id, u.email, u.display_name, u.password_hash, u.global_role, u.avatar_url,
		        u.is_active, u.force_password_change, u.max_projects, u.max_namespaces, u.last_login_at, u.created_at, u.updated_at
		 FROM users u
		 JOIN project_members pm1 ON pm1.user_id = u.id
		 JOIN project_members pm2 ON pm2.project_id = pm1.project_id
		 WHERE pm2.user_id = $1
		   AND u.id != $1
		   AND u.is_active = true
		   AND ($2 = '%%' OR u.email ILIKE $2 OR u.display_name ILIKE $2)
		 ORDER BY u.display_name ASC
		 LIMIT 20`, callerID, q)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var passwordHash sql.NullString
		var avatarURL sql.NullString
		var maxProjects sql.NullInt32
		var maxNamespaces sql.NullInt32
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.GlobalRole,
			&avatarURL, &u.IsActive, &u.ForcePasswordChange, &maxProjects, &maxNamespaces, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		if avatarURL.Valid {
			u.AvatarURL = &avatarURL.String
		}
		if maxProjects.Valid {
			v := int(maxProjects.Int32)
			u.MaxProjects = &v
		}
		if maxNamespaces.Valid {
			v := int(maxNamespaces.Int32)
			u.MaxNamespaces = &v
		}
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.Time
		}
		users = append(users, u)
	}
	return users, nil
}

// ListAll returns all users (active and inactive), ordered by display_name.
func (r *UserRepository) ListAll(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, display_name, password_hash, global_role, avatar_url,
		        is_active, force_password_change, max_projects, max_namespaces, last_login_at, created_at, updated_at
		 FROM users
		 ORDER BY display_name ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var passwordHash sql.NullString
		var avatarURL sql.NullString
		var maxProjects sql.NullInt32
		var maxNamespaces sql.NullInt32
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.GlobalRole,
			&avatarURL, &u.IsActive, &u.ForcePasswordChange, &maxProjects, &maxNamespaces, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		if avatarURL.Valid {
			u.AvatarURL = &avatarURL.String
		}
		if maxProjects.Valid {
			v := int(maxProjects.Int32)
			u.MaxProjects = &v
		}
		if maxNamespaces.Valid {
			v := int(maxNamespaces.Int32)
			u.MaxNamespaces = &v
		}
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.Time
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateGlobalRole changes a user's global role.
func (r *UserRepository) UpdateGlobalRole(ctx context.Context, id uuid.UUID, role string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET global_role = $1, updated_at = now() WHERE id = $2`,
		role, id)
	if err != nil {
		return fmt.Errorf("updating global role: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// UpdateIsActive changes a user's active status.
func (r *UserRepository) UpdateIsActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET is_active = $1, updated_at = now() WHERE id = $2`,
		isActive, id)
	if err != nil {
		return fmt.Errorf("updating is_active: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// UpdateMaxProjects sets or clears the per-user project limit.
func (r *UserRepository) UpdateMaxProjects(ctx context.Context, id uuid.UUID, maxProjects *int) error {
	var val sql.NullInt32
	if maxProjects != nil {
		val = sql.NullInt32{Int32: int32(*maxProjects), Valid: true}
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET max_projects = $1, updated_at = now() WHERE id = $2`,
		val, id)
	if err != nil {
		return fmt.Errorf("updating max_projects: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// UpdateMaxNamespaces sets or clears the per-user namespace limit.
func (r *UserRepository) UpdateMaxNamespaces(ctx context.Context, id uuid.UUID, maxNamespaces *int) error {
	var val sql.NullInt32
	if maxNamespaces != nil {
		val = sql.NullInt32{Int32: int32(*maxNamespaces), Valid: true}
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET max_namespaces = $1, updated_at = now() WHERE id = $2`,
		val, id)
	if err != nil {
		return fmt.Errorf("updating max_namespaces: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// CountByRole returns the number of users with a given global role.
func (r *UserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE global_role = $1 AND is_active = true`, role).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users by role: %w", err)
	}
	return count, nil
}

// UpdatePasswordHash sets a user's password hash and force_password_change flag.
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, forceChange bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, force_password_change = $2, updated_at = now() WHERE id = $3`,
		hash, forceChange, id)
	if err != nil {
		return fmt.Errorf("updating password hash: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	var passwordHash sql.NullString
	var avatarURL sql.NullString
	var maxProjects sql.NullInt32
	var maxNamespaces sql.NullInt32
	var lastLoginAt sql.NullTime

	err := row.Scan(
		&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.GlobalRole,
		&avatarURL, &u.IsActive, &u.ForcePasswordChange, &maxProjects, &maxNamespaces, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}

	u.PasswordHash = passwordHash.String
	if avatarURL.Valid {
		u.AvatarURL = &avatarURL.String
	}
	if maxProjects.Valid {
		v := int(maxProjects.Int32)
		u.MaxProjects = &v
	}
	if maxNamespaces.Valid {
		v := int(maxNamespaces.Int32)
		u.MaxNamespaces = &v
	}
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}

	return &u, nil
}
