package interfaces

import (
	"context"
)

type ILoyaltyRepository interface {
	// Возвращает текущий баланс и сумму списанных баллов
	Balance(ctx context.Context, login string) (float64, float64, error)
	// Проверяет баланс и изменяет его, если достаточно средств
	ChangeLoyaltyPoints(ctx context.Context, login string, order string, num float64) error
	CheckUserOrders(ctx context.Context, login string, num string) (bool, error)
	SendingData(ctx context.Context, login string, number string) error
}
