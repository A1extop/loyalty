package usecase


import (
	"context"
	"errors"
	"strconv"
	"github.com/A1extop/loyalty/internal/services/orders/models"
	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/services/orders/interfaces"
)


type OrderUsecase struct {
	repo interfaces.IOrderRepository
}


func NewOrderUsecase(repo interfaces.IOrderRepository) interfaces.IOrderCase{
	return &OrderUsecase{repo: repo}
}


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


func (u *OrderUsecase) Load(ctx context.Context, numberString string, login string) (bool, error) { // тут точно параша получилась
	ex := validNumber(numberString)
	if !ex {
		return false, errors.Join(errors.New("invalid order number"), domain.ErrUnprocessableEntity)
	}
	exists, err := u.repo.CheckUserOrders(ctx, login, numberString)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	err = u.repo.SendingData(ctx, login, numberString)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (u *OrderUsecase) GetOrders(ctx context.Context, login string) ([]models.History, error) {
	history, err := u.repo.Orders(ctx, login)
	if err != nil {
		return nil, errors.Join(err, domain.ErrInternal)
	}
	return history, nil
}

func (u *OrderUsecase) GetWithdrawals(ctx context.Context, login string) ([]models.History, error) { // тут потом распаковка запаковка, учитывать надо модель
	data, err := u.repo.Orders(ctx, login)
	if err != nil {
		return nil, errors.Join(err, domain.ErrInternal)
	}
	return data, nil
}