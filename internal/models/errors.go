package models

// ErrorResponse is returned as the JSON body for all 4xx and 5xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}
