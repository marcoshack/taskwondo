package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/marcoshack/trackforge/internal/model"
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
		        is_active, last_login_at, created_at, updated_at
		 FROM users WHERE email = $1`, email)

	return scanUser(row)
}

// GetByID returns a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, password_hash, global_role, avatar_url,
		        is_active, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`, id)

	return scanUser(row)
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, display_name, password_hash, global_role, avatar_url, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.DisplayName, sql.NullString{String: user.PasswordHash, Valid: user.PasswordHash != ""},
		user.GlobalRole, user.AvatarURL, user.IsActive)
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

// Search returns active users whose email or display_name match the query (ILIKE).
// Results are limited to 20.
func (r *UserRepository) Search(ctx context.Context, query string) ([]model.User, error) {
	q := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, display_name, password_hash, global_role, avatar_url,
		        is_active, last_login_at, created_at, updated_at
		 FROM users
		 WHERE is_active = true
		   AND (email ILIKE $1 OR display_name ILIKE $1)
		 ORDER BY display_name ASC
		 LIMIT 20`, q)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var passwordHash sql.NullString
		var avatarURL sql.NullString
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.GlobalRole,
			&avatarURL, &u.IsActive, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		if avatarURL.Valid {
			u.AvatarURL = &avatarURL.String
		}
		if lastLoginAt.Valid {
			u.LastLoginAt = &lastLoginAt.Time
		}
		users = append(users, u)
	}
	return users, nil
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	var passwordHash sql.NullString
	var avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := row.Scan(
		&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.GlobalRole,
		&avatarURL, &u.IsActive, &lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
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
	if lastLoginAt.Valid {
		u.LastLoginAt = &lastLoginAt.Time
	}

	return &u, nil
}
