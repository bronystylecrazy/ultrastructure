package web

// OpenAPIErrorResponse is the default schema used by error response presets.
type OpenAPIErrorResponse struct {
	Error   string `json:"error" example:"Internal server error"`
	Code    string `json:"code,omitempty" example:"INTERNAL_ERROR"`
	Message string `json:"message,omitempty" example:"An unexpected error occurred"`
}
