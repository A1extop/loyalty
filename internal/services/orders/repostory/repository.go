package repostory

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/A1extop/loyalty/internal/db"
	"github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/services/orders/interfaces"
	"github.com/A1extop/loyalty/internal/services/orders/models"
)

type OrderRepository struct {
	db *db.Database
}

func NewOrderRepo(db *db.Database) interfaces.IOrderRepository {
	return &OrderRepository{db: db}
}

func (s *OrderRepository) SendingData(ctx context.Context, login string, number string) error {
	query := `INSERT INTO order_history  (username, order_number, status) VALUES ($1, $2, $3)`
	_, err := s.db.Pool.Exec(ctx, query, login, number, "PROCESSING")
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	return nil
}

func (s *OrderRepository) CheckUserOrders(ctx context.Context, login string, num string) (bool, error) {
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

func (s *OrderRepository) Orders(ctx context.Context, login string) ([]models.History, error) {
	slHistory := make([]models.History, 0)
	query := `SELECT order_number, status, accrual, withdrawals, processed_at 
              FROM order_history WHERE username  = $1 ORDER BY processed_at DESC`
	rows, err := s.db.Pool.Query(ctx, query, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var withdrawals int
	var order string
	var status sql.NullString
	var accrual int
	var timeStamp sql.NullTime

	for rows.Next() {
		if err := rows.Scan(&order, &status, &accrual, &withdrawals, &timeStamp); err != nil {
			return nil, err
		}
		accFloat := float64(accrual) / 100
		withdrawFloat := float64(withdrawals) / 100
		statusValue := "PROCESSING"
		if status.Valid && status.String != "" {
			statusValue = status.String
		}
		var timeStampValue time.Time
		if timeStamp.Valid {
			timeStampValue = timeStamp.Time
		}

		history := models.History{
			Order:       order,
			Username:    login,
			Status:      statusValue,
			Accrual:     accFloat,
			Withdrawals: withdrawFloat,
			Uploaded:    timeStampValue,
		}
		slHistory = append(slHistory, history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return slHistory, nil

}
