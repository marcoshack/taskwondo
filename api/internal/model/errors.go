package model

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrUnauthorized       = errors.New("unauthorized")
)
