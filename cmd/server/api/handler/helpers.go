//go:build onnxgenai

package handler

import (
	"encoding/json"
	"net/http"
)

// writeJSONError sends a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
