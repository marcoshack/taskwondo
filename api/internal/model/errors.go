package model

import (
	"errors"
	"fmt"
)

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
	ErrNamespacesDisabled      = errors.New("namespaces are disabled")
	ErrNamespaceNotEmpty       = errors.New("namespace is not empty")
)

// KeyedError wraps an error with a stable error key and interpolation params
// for frontend i18n. The key is a machine-readable identifier (e.g.
// "namespace_slug_reserved") and params provide dynamic values for the
// localized message template.
type KeyedError struct {
	Err    error
	Key    string
	Params map[string]string
}

func (e *KeyedError) Error() string { return e.Err.Error() }
func (e *KeyedError) Unwrap() error { return e.Err }

// NewKeyedError creates a KeyedError wrapping a sentinel with a formatted message.
func NewKeyedError(sentinel error, key, msg string, params map[string]string) *KeyedError {
	return &KeyedError{
		Err:    fmt.Errorf("%s: %w", msg, sentinel),
		Key:    key,
		Params: params,
	}
}

// ErrorKey extracts the error key from an error if it is a KeyedError.
func ErrorKey(err error) (string, map[string]string) {
	var ke *KeyedError
	if errors.As(err, &ke) {
		return ke.Key, ke.Params
	}
	return "", nil
}
