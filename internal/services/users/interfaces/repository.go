package interfaces

import (
	"context"
)

// пока не знаю, как работать с множеством интерфейсов, поэтому посую их сюда же
type IUserRepository interface {
	// Добавляет в таблицу users пользователя, а также создаёт запись в loyalty_accounts
	AddUsers(ctx context.Context, login, password string) error
	// Проверяет наличие данного логина
	UserExists(ctx context.Context, login string) (bool, error)
	// Проверяет наличие и корректность данных, отправленных клиентом
	CheckAvailability(ctx context.Context, login, password string) error
}
