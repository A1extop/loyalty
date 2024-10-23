package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	domain "github.com/A1extop/loyalty/internal/domain"
	"github.com/A1extop/loyalty/internal/hash"
	json2 "github.com/A1extop/loyalty/internal/json"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Store struct {
	db *sql.DB
}
type UserStorage interface {
	// Добавляет в таблицу users пользователя, а также создаёт запись в loyalty_accounts
	AddUsers(login string, password string) error
	// Проверяет наличие данного логина
	UserExists(login string) (bool, error)
	// Проверяет наличие и корректность данных, отправленных клиентом
	CheckAvailability(login string, password string) error
}
type OrderStorage interface {
	// Отправление заказа
	SendingData(login string, number string) error
	// Проверяет заказ, существует ли он и у кого находится
	CheckUserOrders(login string, num string) (bool, error)
	// Получает данные из order_history и возвращает массив структур
	Orders(login string) ([]json2.History, error)
	// Получение заказов со статусом, требующим обновления
	GetOrdersForProcessing() ([]string, error)
	// Обновление данных о заказе в базе данных
	UpdateOrderInDB(orderNumber, status string, accrual int) error
}
type LoyaltyStorage interface {
	// Возвращает текущий баланс и сумму списанных баллов
	Balance(login string) (float64, float64, error)
	// Проверяет баланс и изменяет его, если достаточно средств
	ChangeLoyaltyPoints(login string, order string, num float64) error
}
type OrderProcessingStorage interface {
	// Обновление данных в таблице
	Send(result json2.OrderResponse) error
}

type Storage interface {
	OrderProcessingStorage
	LoyaltyStorage
	OrderStorage
	UserStorage
}

func (s *Store) UpdateOrderInDB(orderNumber, status string, accrual int) error {
	tx, err := s.db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	query := `UPDATE order_history SET status = $1, accrual = $2, processed_at = NOW() WHERE order_number = $3`

	_, err = tx.Exec(query, status, accrual, orderNumber)
	if err != nil {
		return err
	}
	var login string
	query2 := `SELECT username FROM order_history WHERE order_number = $1`
	err = tx.QueryRow(query2, orderNumber).Scan(&login)
	if err != nil {
		return err
	}
	query1 := `UPDATE loyalty_accounts SET current = current + $1 WHERE username = $2`
	_, err = tx.Exec(query1, accrual, login)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// Получение заказов со статусом, требующим обновления
func (s *Store) GetOrdersForProcessing() ([]string, error) {
	query := `SELECT order_number FROM order_history WHERE status NOT IN ('PROCESSED', 'INVALID')`
	rows, err := s.db.Query(query)
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
func (s *Store) SendingData(login string, number string) error {
	query := `INSERT INTO order_history  (username, order_number, status) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(query, login, number, "PROCESSING")
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	// Начало логирования всех записей из order_history
	rows, err := s.db.Query("SELECT order_number, withdrawals FROM order_history")
	if err != nil {
		log.Printf("Ошибка при выборке данных из order_history: %v", err)
		return err
	}
	defer rows.Close()

	log.Println("Содержимое таблицы order_history(проверка sendingData):")
	for rows.Next() {
		var orderNumber string
		var withdrawals int
		err := rows.Scan(&orderNumber, &withdrawals)
		if err != nil {
			log.Printf("Ошибка при сканировании строки: %v", err)
			return err
		}
		log.Printf("order_number: %s, withdrawals: %d", orderNumber, withdrawals)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Ошибка после итерации по строкам: %v", err)
		return err
	}
	// Конец логирования
	return nil
}

func (s *Store) Balance(login string) (float64, float64, error) {
	query := `SELECT current, withdrawn FROM loyalty_accounts WHERE username = $1`
	var current int
	var withdrawn int
	err := s.db.QueryRow(query, login).Scan(&current, &withdrawn)
	if err != nil {
		return 0, 0, err
	}
	currentFloat := float64(current) / 100
	withdrawnFloat := float64(withdrawn) / 100
	return currentFloat, withdrawnFloat, nil
}

func (s *Store) Orders(login string) ([]json2.History, error) {
	slHistory := make([]json2.History, 0)
	query := `SELECT order_number, status, accrual, withdrawals, processed_at 
              FROM order_history WHERE username  = $1 ORDER BY processed_at DESC`
	rows, err := s.db.Query(query, login)
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

		history := json2.History{
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

func (s *Store) CheckUserOrders(login string, num string) (bool, error) {
	query := `SELECT username FROM order_history WHERE order_number = $1`
	var username string
	err := s.db.QueryRow(query, num).Scan(&username)
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

func (s *Store) ChangeLoyaltyPoints(login string, order string, sum float64) error {

	tx, err := s.db.Begin()
	if err != nil {
		errors.Join(err, domain.ErrInternal)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	var current, withdrawn int
	query := `SELECT current, withdrawn 
        FROM loyalty_accounts 
        WHERE username = $1`
	err = tx.QueryRow(query, login).Scan(&current, &withdrawn)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.Join(errors.New("account not found"), domain.ErrNotFound)
		}
		return errors.Join(err, domain.ErrInternal)
	}
	balanceFloat := float64(current) / 100

	if balanceFloat < sum {
		return errors.Join(errors.New("insufficient funds"), domain.ErrPaymentRequired)
	}
	query1 := `SELECT withdrawals FROM order_history WHERE order_number = $2`
	var withdrawals int
	row := tx.QueryRow(query1, order)
	err = row.Scan(&withdrawals)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.Join(errors.New("order not found"), domain.ErrNotFound) //!!!!!здесь появляется ошибка, которой быть не должно, не знаю, что с этим делать

		}
		return errors.Join(err, domain.ErrNotFound)
	}

	if withdrawals != 0 {
		return errors.Join(errors.New("there has already been a write-off for this order"), domain.ErrUnprocessableEntity) //проверка здесь происходит на то, не списывались ли в счёт этого заказа уже баллы, возвращаю 422, но не уверен
	}

	_, err = tx.Exec("UPDATE order_history SET withdrawals = $1 WHERE username = $2 AND order_number = $3", sum*100, login, order)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	newBalance := balanceFloat - sum
	_, err = tx.Exec("UPDATE loyalty_accounts SET current = $1, withdrawn = withdrawn + $2 WHERE username = $3", newBalance*100, sum*100, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	err = tx.Commit()
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	return nil
}

func (s *Store) UserExists(login string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.QueryRow(query, login).Scan(&exists)
	return exists, err
}
func (s *Store) AddUsers(login string, password string) error {

	tx, err := s.db.Begin()
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	insertUserQuery := "INSERT INTO users (username, password_hash) VALUES ($1, $2)"
	_, err = tx.Exec(insertUserQuery, login, password)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	insertLoyaltyAccountQuery := "INSERT INTO loyalty_accounts (username) VALUES ($1)"
	_, err = tx.Exec(insertLoyaltyAccountQuery, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	return nil
}

func (s *Store) CheckAvailability(login string, password string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.QueryRow(query, login).Scan(&exists)
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
	query1 := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND password_hash =$2)"
	err = s.db.QueryRow(query1, login, hashedPassword).Scan(&exists)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	if !exists {
		return errors.Join(errors.New("incorrect login/password pair"), domain.ErrUnauthorized)
	}
	return nil
}

func (s *Store) Send(result json2.OrderResponse) error {
	query := `UPDATE order_history SET status  = $1, accrual = accrual+ $2 WHERE  = order_number = $3`

	tx, err := s.db.Begin()
	go func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, result.Status, result.Accrual*100, result.Order)
	if err != nil {
		return err
	}
	query1 := `UPDATE loyalty_accounts SET current = current + $1 WHERE order_number = $2`
	_, err = tx.Exec(query1, result.Accrual*100, result.Order)
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
func tableExists(db *sql.DB, tableName string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '%s');", tableName)
	err := db.QueryRow(query).Scan(&exists)
	return exists, err
}

func CreateOrConnectTable(db *sql.DB) {

	exists, err := tableExists(db, "users")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE users (
    username VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL
);`)
		if err != nil {
			log.Fatal("database creation error1:", err)
		}
	}

	exists, err = tableExists(db, "order_history")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE order_history (
			order_number VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			status VARCHAR(30) DEFAULT 'REGISTERED',
			accrual INTEGER  DEFAULT 0,
			withdrawals INTEGER DEFAULT 0,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (username) REFERENCES users(username),
			CONSTRAINT unique_user_order UNIQUE (username, order_number)
		);`)
		if err != nil {
			log.Fatal("database creation error2:", err)
		}
	}
	exists, err = tableExists(db, "loyalty_accounts")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE loyalty_accounts (
    username VARCHAR(255) PRIMARY KEY,
    current INTEGER DEFAULT 0,
	withdrawn INTEGER DEFAULT 0,
    FOREIGN KEY (username) REFERENCES users(username)
);`)
		if err != nil {
			log.Fatal("database creation error3:", err)
		}
	}
}

func ConnectDB(connectionToBD string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionToBD)
	if err != nil {
		return nil, err
	}
	return db, nil
}
