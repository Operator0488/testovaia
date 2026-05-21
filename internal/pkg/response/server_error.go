package response

import "net/http"

const (
	InternalMessage        = "An internal error occurred"
	TooManyRequestsMessage = "Too many requests"
)

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
		message:    InternalMessage,
		httpStatus: http.StatusInternalServerError,
		cause:      cause,
	}
}

func TooManyRequests() HTTPError {
	return &serverError{
		message:    TooManyRequestsMessage,
		httpStatus: http.StatusTooManyRequests,
	}
}
