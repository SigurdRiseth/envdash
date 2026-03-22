package models

// ErrorResponse is returned as the JSON body for all 4xx and 5xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationError is returned when a request contains invalid or missing fields.
// Services and handlers can use errors.As to distinguish it from other errors
// and map it to HTTP 400 Bad Request.
type ValidationError struct {
	Message string
}

// Error implements the error interface, returning the validation message.
func (e *ValidationError) Error() string { return e.Message }
