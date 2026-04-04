package server

import (
	"encoding/json"
	"log"
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

// InternalErrorResponse logs the internal error and returns a generic message to the client.
func InternalErrorResponse(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	JSONResponse(w, http.StatusInternalServerError, map[string]string{
		"error": "Internal server error",
	})
}

// SuccessResponse writes a success JSON response.
func SuccessResponse(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// CreatedResponse writes a 201 Created JSON response.
func CreatedResponse(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// DecodeJSON reads and decodes JSON from the request body with a size limit.
// It enforces a maximum body size of 1MB to prevent memory exhaustion.
const maxJSONBodySize = 1 << 20 // 1MB

func DecodeJSON(r *http.Request, v interface{}) error {
	reader := http.MaxBytesReader(nil, r.Body, maxJSONBodySize)
	defer reader.Close()
	return json.NewDecoder(reader).Decode(v)
}
