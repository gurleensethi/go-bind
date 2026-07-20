package gobind

import "net/http"

func NewNotFoundError[T any](v T) *Error[T] {
	return &Error[T]{
		HTTPStatus: http.StatusNotFound,
		Value: v,
	}
}
