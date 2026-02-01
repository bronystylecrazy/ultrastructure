package web

type ErrorDetail struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type Error struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}
