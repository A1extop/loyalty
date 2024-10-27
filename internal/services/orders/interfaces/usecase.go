package interfaces

import (
	"context"

	"github.com/A1extop/loyalty/internal/services/orders/models"
)

type IOrderCase interface {
	Load(ctx context.Context, numberString string, login string) (bool, error)
	GetOrders(ctx context.Context, login string) ([]models.History, error)
	GetWithdrawals(ctx context.Context, login string) ([]models.History, error)
}
