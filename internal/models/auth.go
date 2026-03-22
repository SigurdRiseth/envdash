package models

// AuthRequest is the body accepted by POST /auth/.
type AuthRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AuthCreateResponse is returned by POST /auth/.
// CreatedAt is the server-assigned key creation timestamp in "20060102 15:04" format.
type AuthCreateResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}
