package model

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationToken represents a pending email verification for user registration.
type EmailVerificationToken struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	TokenHash   string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}
