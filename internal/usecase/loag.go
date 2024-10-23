package usecase

import (
	"errors"
	"net/http"
	"strconv"

	domain "github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/store"
)

// Проверка номера на алгоритм Луна
func validNumber(numberStr string) bool {
	var sum int
	alt := false
	for i := len(numberStr) - 1; i >= 0; i-- {
		num, err := strconv.Atoi(string(numberStr[i]))
		if err != nil {
			return false
		}
		if alt {
			num *= 2
			if num > 9 {
				num -= 9
			}
		}
		sum += num
		alt = !alt
	}
	return sum%10 == 0
}

func Load(storage store.Storage, numberString string, login string) (int, error) {

	ex := validNumber(numberString)
	if !ex {
		return http.StatusUnprocessableEntity, errors.New("invalid order number")
	}
	exists, err := storage.CheckUserOrders(login, numberString)
	if err != nil {
		return domain.StatusDetermination(err), err
	}
	if exists {
		return http.StatusOK, nil
	}
	err = storage.SendingData(login, numberString)
	if err != nil {
		return domain.StatusDetermination(err), err
	}
	return http.StatusAccepted, nil
}
