package interfaces

import (
	"context"

	"github.com/A1extop/loyalty/internal/services/users/models"
)

// пока не знаю, как работать с множеством интерфейсов, поэтому посую их сюда же
type IUserCase interface {
	AddAccount(ctx context.Context, user *models.UserCredentials) error
	AuthenticationAccount(ctx context.Context, user *models.UserCredentials) error
}
