package model

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrAlreadyExists      = errors.New("already exists")
	ErrValidation         = errors.New("validation error")
	ErrInvalidTransition  = errors.New("invalid transition")
	ErrOAuthAccountLinked      = errors.New("oauth account already linked to another user")
	ErrStatusIncompatible      = errors.New("status incompatible with target workflow")
	ErrEmbeddingUnavailable    = errors.New("embedding service unavailable")
	ErrFeatureDisabled         = errors.New("feature is disabled")
)
