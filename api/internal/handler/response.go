package handler

import (
	"encoding/json"
	"net/http"
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

// writeError writes an error response following the API error format.
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
