package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/marcoshack/taskwondo/internal/model"
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
