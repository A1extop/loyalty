package usecase

import (
	"net/http"

	errors2 "github.com/A1extop/loyalty/internal/errors"
	"github.com/A1extop/loyalty/internal/store"
)

func WriteOff(storage store.Storage, login string, order string, sum float64) (int, error) {
	err := storage.ChangeLoyaltyPoints(login, order, sum)
	if err != nil {
		return errors2.StatusDetermination(err), err
	}
	return http.StatusOK, nil
}
