package response

// Envelope — обёртка для разбора ответов (payload и/или error).
type Envelope struct {
	Payload any        `json:"payload,omitempty"`
	Error   *ErrorPart `json:"error,omitempty"`
}

// ErrorPart — тело поля error в JSON.
type ErrorPart struct {
	TraceID string              `json:"traceId"`
	Message string              `json:"message"`
	Detail  map[string][]string `json:"detail,omitempty"`
}
