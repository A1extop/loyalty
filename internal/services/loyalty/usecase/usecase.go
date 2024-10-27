package usecase

import (
	"context"

	"github.com/A1extop/loyalty/internal/services/loyalty/interfaces"
)

type LoyaltyUsecase struct {
	repo interfaces.ILoyaltyRepository
}

func NewLoyaltyUsecase(repo interfaces.ILoyaltyRepository) interfaces.ILoyaltyCase {
	return &LoyaltyUsecase{repo: repo}
}

func (u *LoyaltyUsecase) GetBalanceAccount(ctx context.Context, login string) (float64, float64, error) {
	return u.repo.Balance(ctx, login)
}

func (u *LoyaltyUsecase) WriteOff(ctx context.Context, login string, order string, sum float64) error {
	ex, err := u.repo.CheckUserOrders(ctx, login, order)
	if err != nil {
		return err
	}
	if !ex {
		err := u.repo.SendingData(ctx, login, order)
		if err != nil {
			return err
		}
	}
	err = u.repo.ChangeLoyaltyPoints(ctx, login, order, sum)
	if err != nil {
		return err
	}
	return nil
}
