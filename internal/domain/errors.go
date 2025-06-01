package domain

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized = errors.New("user is not authorized")

	ErrInternal            = errors.New("internal server error")
	ErrConflict            = errors.New("conflict")
	ErrUnprocessableEntity = errors.New("unprocessable entity") //
	ErrTooManyRequests     = errors.New("too many requests")
	ErrPaymentRequired     = errors.New("status Payment Required")
	ErrNotFound            = errors.New("status Payment Required")
	ErrUserNotFound        = errors.New("user not found")
	ErrNoContent           = errors.New("no data")
)

func StatusDetermination(err error) int {
	switch {
	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrUnprocessableEntity):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrTooManyRequests):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrPaymentRequired):
		return http.StatusPaymentRequired
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrNoContent):
		return http.StatusNoContent
	}
	return http.StatusOK
}
