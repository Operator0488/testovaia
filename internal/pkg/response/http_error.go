package response

// HTTPError — ошибка с HTTP-статусом и сообщением для клиента.
type HTTPError interface {
	error
	HTTPStatus() int
	Code() string
	Message() string
}
