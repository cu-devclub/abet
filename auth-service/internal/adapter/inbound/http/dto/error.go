package dto

// ErrorResponse is the JSON body of every error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
