package response

import "net/http"

type clientError struct {
	message    string
	httpStatus int
}

func (e *clientError) Error() string   { return e.message }
func (e *clientError) HTTPStatus() int { return e.httpStatus }
func (e *clientError) Code() string    { return "" }
func (e *clientError) Message() string { return e.message }

func NotFound() HTTPError {
	return &clientError{
		message:    "resource not found",
		httpStatus: http.StatusNotFound,
	}
}

func BadRequest(message string) HTTPError {
	return &clientError{
		message:    message,
		httpStatus: http.StatusBadRequest,
	}
}

func Conflict(message string) HTTPError {
	return &clientError{
		message:    message,
		httpStatus: http.StatusConflict,
	}
}

func Forbidden(message string) HTTPError {
	return &clientError{
		message:    message,
		httpStatus: http.StatusForbidden,
	}
}

func Unauthorized() HTTPError {
	return &clientError{
		message:    "Authentication required",
		httpStatus: http.StatusUnauthorized,
	}
}

func Locked(message string) HTTPError {
	return &clientError{
		message:    message,
		httpStatus: http.StatusLocked,
	}
}
