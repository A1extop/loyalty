package repostory

import (
	"context"

	"github.com/A1extop/loyalty/internal/db"
	"github.com/A1extop/loyalty/internal/services/systemloyalty/interfaces"
)

type SystemLoyaltyRepository struct {
	db *db.Database
}

func NewSystemLoyaltyRepo(db *db.Database) interfaces.ISystemLoyaltyRepository {
	return &SystemLoyaltyRepository{db: db}
}

func (s *SystemLoyaltyRepository) GetOrdersForProcessing(ctx context.Context) ([]string, error) {
	query := `SELECT order_number FROM order_history WHERE status NOT IN ('PROCESSED', 'INVALID')`
	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []string
	for rows.Next() {
		var orderNumber string
		if err := rows.Scan(&orderNumber); err != nil {

			return nil, err
		}
		orders = append(orders, orderNumber)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *SystemLoyaltyRepository) UpdateOrderInDB(ctx context.Context, orderNumber, status string, accrual int) error {
	tx, err := s.db.Pool.Begin(ctx)
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()
	query := `UPDATE order_history SET status = $1, accrual = $2, processed_at = NOW() WHERE order_number = $3`

	_, err = tx.Exec(ctx, query, status, accrual, orderNumber)
	if err != nil {
		return err
	}
	var login string
	query2 := `SELECT username FROM order_history WHERE order_number = $1`
	err = tx.QueryRow(ctx, query2, orderNumber).Scan(&login)
	if err != nil {
		return err
	}
	query1 := `UPDATE loyalty_accounts SET current = current + $1 WHERE username = $2`
	_, err = tx.Exec(ctx, query1, accrual, login)
	if err != nil {
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}
