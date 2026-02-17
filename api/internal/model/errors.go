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
)
