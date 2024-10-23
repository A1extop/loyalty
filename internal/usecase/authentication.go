package usecase

import (
	"net/http"

	errors2 "github.com/A1extop/loyalty/internal/errors"
	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/A1extop/loyalty/internal/store"
)

func AuthenticationAccount(storage store.Storage, user *json2.UserCredentials) (int, error) {
	err := storage.CheckAvailability(user.Login, user.Password)
	if err != nil {
		status := errors2.StatusDetermination(err)
		return status, err
	}
	return http.StatusOK, nil
}
