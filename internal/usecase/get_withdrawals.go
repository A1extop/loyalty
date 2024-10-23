package usecase

import (
	"net/http"

	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/A1extop/loyalty/internal/store"
)

func GetWithdrawals(storage store.Storage, login string) (int, []byte, error) {
	data, err := storage.Orders(login)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	dataByte, err := json2.PackingWithdrawalsJSON(data)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, dataByte, nil
}
