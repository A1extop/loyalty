package interfaces

import (
	"context"

	"github.com/A1extop/loyalty/internal/services/orders/models"
)

type IOrderRepository interface {
	// Отправление заказа
	SendingData(ctx context.Context, login string, number string) error
	// Проверяет заказ, существует ли он и у кого находится
	CheckUserOrders(ctx context.Context, login string, num string) (bool, error)
	// Получает данные из order_history и возвращает массив структур
	Orders(ctx context.Context, login string) ([]models.History, error)
}
