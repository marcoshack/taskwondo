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
	InviteCode  string // optional project invite code to auto-accept after verification
	ExpiresAt   time.Time
	CreatedAt   time.Time
}
