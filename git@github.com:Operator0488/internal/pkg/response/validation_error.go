package response

import "net/http"

const (
	Message = "invalid request parameters"
	Code    = "UNPROCESSABLE_ENTITY"
)

type validationError struct {
	details map[string][]string
}

func (e *validationError) Error() string                { return Message }
func (e *validationError) HTTPStatus() int              { return http.StatusUnprocessableEntity }
func (e *validationError) Code() string                 { return Code }
func (e *validationError) Message() string              { return Message }
func (e *validationError) Details() map[string][]string { return e.details }

func UnprocessableEntity(fieldErrors map[string][]string) HTTPError {
	return &validationError{
		details: fieldErrors,
	}
}
