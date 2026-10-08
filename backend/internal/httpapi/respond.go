package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxBodyBytes = 4 << 10

type apiError struct {
	status  int
	code    string
	message string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, e apiError) {
	type body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	writeJSON(w, e.status, map[string]body{"error": {Code: e.code, Message: e.message}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) *apiError {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return &apiError{http.StatusRequestEntityTooLarge, "body_too_large", "Request body is too large"}
		}
		return &apiError{http.StatusBadRequest, "invalid_json", "Request body must be a JSON object"}
	}
	return nil
}

var errNotFound = apiError{http.StatusNotFound, "not_found", "Not found"}
