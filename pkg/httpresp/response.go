package httpresp

import (
	"context"
	"net/http"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/internal/pkg/response"
)

// Response — универсальный результат обработки до записи в HTTP.
type Response[T any] struct {
	data       T
	statusCode int
	err        error
}

func Ok[T any](data T) Response[T] {
	return Response[T]{data: data, statusCode: http.StatusOK}
}

func Created[T any](data T) Response[T] {
	return Response[T]{data: data, statusCode: http.StatusCreated}
}

func NoContent[T any]() Response[T] {
	return Response[T]{statusCode: http.StatusNoContent}
}

func Accepted[T any](data T) Response[T] {
	return Response[T]{data: data, statusCode: http.StatusAccepted}
}

func BadRequest[T any](message string) Response[T] {
	return Response[T]{err: response.BadRequest(message)}
}

func Conflict[T any](message string) Response[T] {
	return Response[T]{err: response.Conflict(message)}
}

func Forbidden[T any](message string) Response[T] {
	return Response[T]{err: response.Forbidden(message)}
}

func NotFound[T any]() Response[T] {
	return Response[T]{err: response.NotFound()}
}

func Unauthorized[T any]() Response[T] {
	return Response[T]{err: response.Unauthorized()}
}

func Internal[T any](cause error) Response[T] {
	return Response[T]{err: response.Internal(cause)}
}

func TooManyRequests[T any]() Response[T] {
	return Response[T]{err: response.TooManyRequests()}
}

func (r Response[T]) IsErr() bool { return r.err != nil }

func (r Response[T]) Err() error { return r.err }

func (r Response[T]) Data() T { return r.data }

func (r Response[T]) StatusCode() int {
	if r.err != nil {
		return 0
	}
	if r.statusCode == 0 {
		return http.StatusOK
	}

	return r.statusCode
}

func Handle[T any](ctx context.Context, w http.ResponseWriter, r *http.Request, resp Response[T]) {
	if resp.IsErr() {
		response.WriteError(ctx, w, r, resp.Err())

		return
	}

	code := resp.StatusCode()
	if code == http.StatusNoContent {
		w.WriteHeader(code)

		return
	}

	response.WriteJSON(w, code, resp.Data())
}

func JSONErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	response.JSONErrorHandler(w, r, err)
}
