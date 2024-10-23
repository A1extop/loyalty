package usecase

import (
	"net/http"

	domain "github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/store"
)

func WriteOff(storage store.Storage, login string, order string, sum float64) (int, error) {
	ex, err := storage.CheckUserOrders(login, order)
	if err != nil {
		return http.StatusInternalServerError, domain.ErrInternal
	}
	if !ex {
		err := storage.SendingData(login, order)
		if err != nil {
			return http.StatusInternalServerError, domain.ErrInternal
		}
	}
	err = storage.ChangeLoyaltyPoints(login, order, sum)
	if err != nil {
		return domain.StatusDetermination(err), err
	}
	return http.StatusOK, nil
}
