package interfaces

import (
	"context"
)

// пока не знаю, как работать с множеством интерфейсов, поэтому посую их сюда же
type ISystemLoyaltyRepository interface {
	// Получение заказов со статусом, требующим обновления
	GetOrdersForProcessing(ctx context.Context) ([]string, error)
	// Обновление данных о заказе в базе данных
	UpdateOrderInDB(ctx context.Context, orderNumber, status string, accrual int) error
}
