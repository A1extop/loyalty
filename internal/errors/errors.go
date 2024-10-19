package domain

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInternal            = errors.New("internal server error")
	ErrConflict            = errors.New("conflict")
	ErrUnprocessableEntity = errors.New("unprocessable entity") //
	ErrTooManyRequests     = errors.New("too many requests")
	ErrPaymentRequired     = errors.New("status Payment Required")
	ErrNotFound            = errors.New("status Payment Required")
)

func StatusDetermination(err error) int {
	if err != nil {
		if errors.Is(err, ErrInternal) {
			return http.StatusInternalServerError
		}
		if errors.Is(err, ErrConflict) {
			return http.StatusConflict
		}
		if errors.Is(err, ErrUnauthorized) {
			return http.StatusUnauthorized
		}
		if errors.Is(err, ErrUnprocessableEntity) {
			return http.StatusUnprocessableEntity
		}
		if errors.Is(err, ErrTooManyRequests) {
			return http.StatusTooManyRequests
		}
		if errors.Is(err, ErrPaymentRequired) {
			return http.StatusPaymentRequired
		}
		if errors.Is(err, ErrNotFound) {
			return http.StatusNotFound
		}

	}
	return http.StatusOK
}
