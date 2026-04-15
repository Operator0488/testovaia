package response

import (
	"net/http"
)

// HTTPError — ошибка с HTTP-статусом и сообщением для клиента.
type HTTPError interface {
	error
	HTTPStatus() int
	Code() string
	Message() string
	Cause() error
}

// Envelope — обёртка для разбора ответов (payload и/или error).
type Envelope struct {
	Payload any        `json:"payload,omitempty"`
	Error   *ErrorPart `json:"error,omitempty"`
}

// ErrorPart — тело поля error в JSON (`trace_id`, `message`).
type ErrorPart struct {
	TraceID string `json:"trace_id"`
	Message string `json:"message"`
}

type clientError struct {
	code       string
	message    string
	httpStatus int
}

func (e *clientError) Error() string   { return e.message }
func (e *clientError) HTTPStatus() int { return e.httpStatus }
func (e *clientError) Code() string    { return e.code }
func (e *clientError) Message() string { return e.message }
func (e *clientError) Cause() error    { return nil }

func NotFound(resource string) HTTPError {
	return &clientError{
		code:       "NOT_FOUND",
		message:    resource + " not found",
		httpStatus: http.StatusNotFound,
	}
}

func BadRequest(message string) HTTPError {
	return &clientError{
		code:       "BAD_REQUEST",
		message:    message,
		httpStatus: http.StatusBadRequest,
	}
}

func Forbidden(message string) HTTPError {
	return &clientError{
		code:       "FORBIDDEN",
		message:    message,
		httpStatus: http.StatusForbidden,
	}
}

func Unauthorized() HTTPError {
	return &clientError{
		code:       "UNAUTHORIZED",
		message:    "Authentication required",
		httpStatus: http.StatusUnauthorized,
	}
}

type serverError struct {
	code       string
	message    string
	httpStatus int
	cause      error
}

func (e *serverError) Error() string   { return e.message }
func (e *serverError) HTTPStatus() int { return e.httpStatus }
func (e *serverError) Code() string    { return e.code }
func (e *serverError) Message() string { return e.message }
func (e *serverError) Cause() error    { return e.cause }

func Internal(cause error) HTTPError {
	return &serverError{
		code:       "INTERNAL_ERROR",
		message:    "An internal error occurred",
		httpStatus: http.StatusInternalServerError,
		cause:      cause,
	}
}

func TooManyRequests() HTTPError {
	return &serverError{
		code:       "TOO_MANY_REQUESTS",
		message:    "Too many requests",
		httpStatus: http.StatusTooManyRequests,
		cause:      nil,
	}
}
