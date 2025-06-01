package repostory

import (
	"context"
	"database/sql"
	"errors"

	"github.com/A1extop/loyalty/internal/db"
	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/services/loyalty/interfaces"
)

type LoyaltyRepository struct {
	db *db.Database
}

func NewLoyaltyRepo(db *db.Database) interfaces.ILoyaltyRepository {
	return &LoyaltyRepository{db: db}
}

func (s *LoyaltyRepository) Balance(ctx context.Context, login string) (float64, float64, error) {
	query := `SELECT current, withdrawn FROM loyalty_accounts WHERE username = $1`
	var current int
	var withdrawn int
	err := s.db.Pool.QueryRow(ctx, query, login).Scan(&current, &withdrawn)
	if err != nil {
		return 0, 0, err
	}
	currentFloat := float64(current) / 100
	withdrawnFloat := float64(withdrawn) / 100
	return currentFloat, withdrawnFloat, nil
}

func (s *LoyaltyRepository) ChangeLoyaltyPoints(ctx context.Context, login string, order string, sum float64) error {
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
	query := `SELECT current, withdrawn 
        FROM loyalty_accounts 
        WHERE username = $1`
	var current int
	var withdrawn int
	err = tx.QueryRow(ctx, query, login).Scan(&current, &withdrawn)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.Join(errors.New("order not found"), domain.ErrNotFound)
		}
		return errors.Join(err, domain.ErrInternal)
	}
	balanceFloat := float64(current) / 100

	if balanceFloat < sum {
		return errors.Join(errors.New("insufficient funds"), domain.ErrPaymentRequired)
	}
	query1 := `SELECT withdrawals FROM order_history WHERE order_number = $1`
	var withdrawals int
	row := tx.QueryRow(ctx, query1, order)
	err = row.Scan(&withdrawals)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.Join(errors.New("order not found"), domain.ErrNotFound)
		}
		return errors.Join(err, domain.ErrInternal)
	}
	if withdrawals != 0 {
		return errors.Join(errors.New("there has already been a write-off for this order"), domain.ErrUnprocessableEntity)
	}

	_, err = tx.Exec(ctx, "UPDATE order_history SET withdrawals = $1 WHERE username = $2 AND order_number = $3", sum*100, login, order)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	newBalance := balanceFloat - sum
	_, err = tx.Exec(ctx, "UPDATE loyalty_accounts SET current = $1, withdrawn = withdrawn + $2 WHERE username = $3", newBalance*100, sum*100, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	return nil
}

func (s *LoyaltyRepository) CheckUserOrders(ctx context.Context, login string, num string) (bool, error) {
	query := `SELECT username FROM order_history WHERE order_number = $1`
	var username string
	err := s.db.Pool.QueryRow(ctx, query, num).Scan(&username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { //её нет тогда false проверяется и даёт зарегать
			return false, nil
		}
		return false, errors.Join(err, domain.ErrInternal)
	}
	if login == username {
		return true, nil // есть и при этом у этого пользователя
	}
	return false, errors.Join(errors.New("Conflict"), domain.ErrConflict)
}

func (s *LoyaltyRepository) SendingData(ctx context.Context, login string, number string) error {
	query := `INSERT INTO order_history  (username, order_number, status) VALUES ($1, $2, $3)`
	_, err := s.db.Pool.Exec(ctx, query, login, number, "PROCESSING")
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	return nil
}
