package usecase

import (
	"errors"
	"net/http"

	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/A1extop/loyalty/internal/store"
)

func GetOrders(storage store.Storage, login string) (int, []byte, error) {
	history, err := storage.Orders(login)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	if len(history) == 0 {
		return http.StatusNoContent, nil, errors.New("no orders")
	}
	data, err := json2.PackingHistoryJSON(history)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, data, nil
}
