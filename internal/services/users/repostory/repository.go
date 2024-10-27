package repostory

import (
	"context"
	"errors"

	"github.com/A1extop/loyalty/internal/db"
	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/hash"
	"github.com/A1extop/loyalty/internal/services/users/interfaces"
)

// //////////////////////////////изменить названия переменных query !!!!!!!
type UserRepository struct {
	db *db.Database
}

func NewUserRepo(db *db.Database) interfaces.IUserRepository {
	return &UserRepository{db: db}
}

func (s *UserRepository) UserExists(ctx context.Context, login string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.Pool.QueryRow(ctx, query, login).Scan(&exists)
	return exists, errors.Join(err, domain.ErrInternal) // а какую ошибку иначе возвращать
}

func (s *UserRepository) AddUsers(ctx context.Context, login string, password string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()
	insertUserQuery := "INSERT INTO users (username, password_hash) VALUES ($1, $2)"
	_, err = tx.Exec(ctx, insertUserQuery, login, password)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	insertLoyaltyAccountQuery := "INSERT INTO loyalty_accounts (username) VALUES ($1)"
	_, err = tx.Exec(ctx, insertLoyaltyAccountQuery, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	return nil
}

func (s *UserRepository) CheckAvailability(ctx context.Context, login string, password string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.Pool.QueryRow(ctx, query, login).Scan(&exists)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	if !exists {
		return errors.Join(errors.New("incorrect login/password pair"), domain.ErrUnauthorized)
	}
	hashedPassword, err := hash.HashPassword(password, "secretKey") // подумаю как протянуть ещё!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	queryExists := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND password_hash =$2)"
	err = s.db.Pool.QueryRow(ctx, queryExists, login, hashedPassword).Scan(&exists)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	if !exists {
		return errors.Join(errors.New("incorrect login/password pair"), domain.ErrUnauthorized)
	}
	return nil
}
