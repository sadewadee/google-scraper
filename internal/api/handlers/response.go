package handlers

import (
	"encoding/json"
	"net/http"
)

// APIError represents an error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PaginatedResponse wraps paginated results
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}

// RenderJSON renders a JSON response
func RenderJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// RenderError renders an error response
func RenderError(w http.ResponseWriter, code int, message string) {
	RenderJSON(w, code, APIError{
		Code:    code,
		Message: message,
	})
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(data interface{}, total, page, perPage int) PaginatedResponse {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
