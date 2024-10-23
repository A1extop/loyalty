package usecase

import (
	"net/http"

	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/A1extop/loyalty/internal/store"
)

func GetBalance(storage store.Storage, login string) (int, []byte, error) {

	current, withdrawn, err := storage.Balance(login)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	balance := json2.NewBalance(current, withdrawn)
	data, err := json2.PackingMoney(*balance)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, data, nil
}
