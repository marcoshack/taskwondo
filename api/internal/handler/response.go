package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
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
