package interfaces

import (
	"context"
)

type ILoyaltyCase interface {
	GetBalanceAccount(ctx context.Context, login string) (float64, float64, error)
	WriteOff(ctx context.Context, login string, order string, sum float64) error
}
