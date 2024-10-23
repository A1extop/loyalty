package usecase

import (
	"errors"
	"net/http"

	"github.com/A1extop/loyalty/internal/hash"
	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/A1extop/loyalty/internal/store"
)

func AddAccount(storage store.Storage, user *json2.UserCredentials) (int, error) {
	exists, err := storage.UserExists(user.Login)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if exists {
		return http.StatusConflict, errors.New("login is already taken")
	}

	hashedPassword, err := hash.HashPassword(user.Password, "secretKey") //"secretKey"вынесу в конфиг
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = storage.AddUsers(user.Login, hashedPassword)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
