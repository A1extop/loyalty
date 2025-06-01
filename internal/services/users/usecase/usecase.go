package usecase

import (
	"context"
	"errors"

	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/hash"
	"github.com/A1extop/loyalty/internal/services/users/interfaces"
	"github.com/A1extop/loyalty/internal/services/users/models"
)

type UserUsecase struct {
	repo interfaces.IUserRepository
}

func NewUserUsecase(repo interfaces.IUserRepository) interfaces.IUserCase {
	return &UserUsecase{repo: repo}
}

func (u *UserUsecase) AddAccount(ctx context.Context, user *models.UserCredentials) error { // изменил, добавил проверку
	exists, err := u.repo.UserExists(ctx, user.Login) //
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrConflict
	}
	hash1 := hash.NewSHA256Hasher()
	hashedPassword, err := hash.HashPassword(user.Password, hash1)
	if err != nil {
		return errors.Join(domain.ErrInternal, errors.New("hashing error"))
	}
	return u.repo.AddUsers(ctx, user.Login, hashedPassword)
}

func (u *UserUsecase) AuthenticationAccount(ctx context.Context, user *models.UserCredentials) error {
	hash1 := hash.NewSHA256Hasher()
	hashedPassword, err := hash.HashPassword(user.Password, hash1)
	if err != nil {
		return domain.ErrInternal
	}
	return u.repo.CheckAvailability(ctx, user.Login, hashedPassword)
}
