package model

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a pending password reset request.
type PasswordResetToken struct {
	ID        uuid.UUID
	Email     string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
