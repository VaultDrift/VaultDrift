package server

import (
	"encoding/json"
	"net/http"
)

// JSONResponse writes a JSON response.
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// ErrorResponse writes an error JSON response.
func ErrorResponse(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, map[string]string{
		"error": message,
	})
}

// SuccessResponse writes a success JSON response.
func SuccessResponse(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}
