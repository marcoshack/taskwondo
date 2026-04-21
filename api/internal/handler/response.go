package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
)

// Error code constants used in API error responses.
const (
	CodeValidationError    = "VALIDATION_ERROR"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeNotFound           = "NOT_FOUND"
	CodeForbidden          = "FORBIDDEN"
	CodeConflict           = "CONFLICT"
	CodeFileTooLarge       = "FILE_TOO_LARGE"
	CodeSMTPError          = "SMTP_ERROR"
	CodeOAuthError         = "OAUTH_ERROR"
	CodeNotConfigured      = "NOT_CONFIGURED"
	CodeNamespacesDisabled = "NAMESPACES_DISABLED"
	CodeStatusIncompatible = "STATUS_INCOMPATIBLE"
	CodeNamespaceNotEmpty  = "NAMESPACE_NOT_EMPTY"
	CodeInvalidTransition  = "INVALID_TRANSITION"
)

// writeJSON writes a raw JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeData writes a success response wrapped in a {"data": ...} envelope.
func writeData(w http.ResponseWriter, status int, data interface{}) {
	writeJSON(w, status, map[string]interface{}{"data": data})
}

// avatarURL converts a stored avatar reference (storage key or external URL)
// into a URL suitable for API responses. Storage keys become avatar endpoint
// URLs with an optional cache-busting version parameter.
func avatarURL(raw *string, userID uuid.UUID, version int64) *string {
	if raw == nil || *raw == "" {
		return nil
	}
	if (*raw)[0] != 'h' {
		url := fmt.Sprintf("/api/v1/users/%s/avatar", userID)
		if version > 0 {
			url += fmt.Sprintf("?v=%d", version)
		}
		return &url
	}
	return raw
}

// writeError writes an error response following the API error format.
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// writeErrorKeyed writes an error response with a stable error_key for frontend i18n.
// The params map provides interpolation values for the localized message template.
func writeErrorKeyed(w http.ResponseWriter, status int, code, errorKey, message string, params map[string]string) {
	resp := map[string]interface{}{
		"code":      code,
		"error_key": errorKey,
		"message":   message,
	}
	if len(params) > 0 {
		resp["params"] = params
	}
	writeJSON(w, status, map[string]interface{}{"error": resp})
}

// writeErrorFromService writes an error response, automatically extracting
// error_key and params from a KeyedError if present.
func writeErrorFromService(w http.ResponseWriter, status int, code string, err error) {
	key, params := model.ErrorKey(err)
	if key != "" {
		writeErrorKeyed(w, status, code, key, err.Error(), params)
		return
	}
	writeError(w, status, code, err.Error())
}

// parseUUIDParam parses a UUID from a chi URL parameter. On failure it writes
// a validation error and returns uuid.Nil, false.
func parseUUIDParam(w http.ResponseWriter, r *http.Request, param, errMsg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, errMsg)
		return uuid.Nil, false
	}
	return id, true
}

// parseOptionalUUID parses an optional UUID string pointer. On failure it
// writes a validation error and returns uuid.Nil, false.
func parseOptionalUUID(w http.ResponseWriter, s *string, errMsg string) (uuid.UUID, bool) {
	id, err := uuid.Parse(*s)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, errMsg)
		return uuid.Nil, false
	}
	return id, true
}

// unmarshalField extracts a typed value from a raw JSON field. Returns the
// value and true on success. On unmarshal failure it writes a validation error
// and returns the zero value and false.
func unmarshalField[T any](w http.ResponseWriter, raw json.RawMessage, field string) (T, bool) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, fmt.Sprintf("invalid value for %s", field))
		var zero T
		return zero, false
	}
	return v, true
}

// unmarshalNullableUUID extracts a nullable UUID field from raw JSON.
// Returns (id, cleared, ok). If the field is JSON null, cleared is true.
// On parse failure it writes a validation error and ok is false.
func unmarshalNullableUUID(w http.ResponseWriter, raw json.RawMessage, field string) (uuid.UUID, bool, bool) {
	if string(raw) == "null" {
		return uuid.Nil, true, true
	}
	var idStr string
	if err := json.Unmarshal(raw, &idStr); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid "+field+" format")
		return uuid.Nil, false, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid "+field+" format")
		return uuid.Nil, false, false
	}
	return id, false, true
}

// unmarshalNullableDate extracts a nullable date (YYYY-MM-DD) field from raw JSON.
// Returns (date, cleared, ok). If the field is JSON null, cleared is true.
func unmarshalNullableDate(w http.ResponseWriter, raw json.RawMessage, field string) (time.Time, bool, bool) {
	if string(raw) == "null" {
		return time.Time{}, true, true
	}
	var dateStr string
	if err := json.Unmarshal(raw, &dateStr); err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid "+field+" format, expected YYYY-MM-DD")
		return time.Time{}, false, false
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, CodeValidationError, "invalid "+field+" format, expected YYYY-MM-DD")
		return time.Time{}, false, false
	}
	return t, false, true
}
