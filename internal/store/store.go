package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/A1extop/loyalty/internal/domain"
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
type Storage interface {
	//Добавляет в таблицу users пользователя, а также создаёт запись в loyalty_accounts транзакциями
	AddUsers(login string, key string, password string) error
	//Проверяет наличие и корректность данных, отправленынх клиентом в таблице
	CheckAvailability(login string, password string) error

	SendingData(login string, number string) error
	//Проверка на наличие номера в таблице
	CheckNumber(number string) error
	//Проверка на наличие пользователя (есть вопрос,а нужен ли этот метод?)))
	CheckUserOrders(login string, num string) (bool, error)
	//Полученает данные с таблицы order_history, заворачивает данные в структуру и возвращает массив структур
	Orders(login string) ([]json2.History, error)
	//Заходит в таблицу loyalt_accounts и возвращает current, withdrawn error
	Balance(login string) (float64, float64, error)
	//Проверяет, хватает ли баланса для списания средств. Если да, то идёт в loyalty_accounts, там изменяет текущий баланс клиента, изменяет также в таблице order_history accrual и withdrawals
	ChangeLoyaltyPoints(login string, order string, num float64) error
	//Получение заказов с нужными статусами и отправка их в канал
	FetchAndUpdateOrderNumbers(orderChan chan<- string)
	//Обновление данных в таблице
	Send(result json2.OrderResponse) error
}

func (s *Store) SendingData(login string, number string) error {
	query := `INSERT INTO orders (username, order_number) VALUES ($1, $2)`
	_, err := s.db.Exec(query, login, number)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
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
	currentFloat := float64(current) / 10
	withdrawnFloat := float64(withdrawn) / 10
	return currentFloat, withdrawnFloat, nil
}

func (s *Store) Orders(login string) ([]json2.History, error) {
	slHistory := make([]json2.History, 0)
	query := `SELECT history_id, order_number, status, accrual, withdrawals, processed_at 
              FROM order_history WHERE username  = $1 ORDER BY processed_at DESC`
	rows, err := s.db.Query(query, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var historyID int
	var withdrawals int
	var order string
	var status string
	var accrual int
	var timeStamp time.Time

	for rows.Next() {
		if err := rows.Scan(&historyID, &order, &status, &accrual, &withdrawals, &timeStamp); err != nil {
			continue
		}
		accFloat := float64(accrual) / 10
		withdrawFloat := float64(withdrawals) / 10

		history := json2.History{
			HistoryID:   historyID,
			Order:       order,
			Username:    login,
			Status:      status,
			Accrual:     accFloat,
			Withdrawals: withdrawFloat,
			Uploaded:    timeStamp,
		}
		slHistory = append(slHistory, history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return slHistory, nil
}

func (s *Store) CheckUserOrders(login string, num string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM orders WHERE username = $1 AND order_number = $2)`

	var exists bool
	err := s.db.QueryRow(query, login, num).Scan(&exists)
	if err != nil {
		return false, errors.Join(err, domain.ErrInternal)
	}
	return exists, nil
}

func (s *Store) CheckNumber(number string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM orders WHERE order_number=$1)"
	err := s.db.QueryRow(query, number).Scan(&exists)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	if exists {

		return errors.Join(errors.New("Conflict"), domain.ErrConflict)
	}
	return nil
}

func (s *Store) ChangeLoyaltyPoints(login string, order string, num float64) error {
	num *= 10
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	var balance int
	query := `SELECT balance FROM loyalty_accounts WHERE username = $1`
	err = s.db.QueryRow(query, login).Scan(&balance)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	balanceFloat := float64(balance)
	if balanceFloat-num < 0 {
		return errors.Join(errors.New("there are insufficient funds in the account"), domain.ErrPaymentRequired)
	}
	query = `SELECT withdrawals FROM orders WHERE username = $1`
	var withdrawals int
	row := tx.QueryRow(query, login)
	err = row.Scan(&withdrawals)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)

	}
	currentBalance := balanceFloat - num
	withdrawalsFloat := float64(withdrawals)
	if withdrawalsFloat != 10.0 {
		return errors.Join(errors.New("there has already been a write-off for this order"), domain.ErrUnprocessableEntity) //проверка здесь происходит на то, не списывались ли в счёт этого заказа уже баллы, возвращаю 422, но не уверен
	}
	_, err = tx.Exec("UPDATE order_history SET withdrawals  = $1 WHERE username = $2", num, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	_, err = tx.Exec("UPDATE loyalty_accounts SET balance = $1 WHERE username = $2", currentBalance, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	err = tx.Commit()
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	return nil
}

func (s *Store) AddUsers(login string, key string, password string) error {

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.QueryRow(query, login).Scan(&exists)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}

	if exists {
		return errors.Join(errors.New("this login is already taken)"), domain.ErrConflict)
	}
	hashedPassword, err := hash.HashPassword(password, key)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
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
	_, err = tx.Exec(insertUserQuery, login, hashedPassword)
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

// Проверяет статусы и заказов, если 'REGISTERED', 'PROCESSING', то добавляет их в массив заказов и отправляет массив
func FetchOrderNumbersFromDB(db *sql.DB) ([]string, error) {
	query := `SELECT order_number FROM order_history WHERE status IN ('REGISTERED', 'PROCESSING')`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orderNumbers []string
	for rows.Next() {
		var orderNumber string
		if err := rows.Scan(&orderNumber); err != nil {
			return nil, err
		}
		orderNumbers = append(orderNumbers, orderNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orderNumbers, nil
}

func (s *Store) FetchAndUpdateOrderNumbers(orderChan chan<- string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		orderNumbers, err := FetchOrderNumbersFromDB(s.db)
		if err != nil {
			log.Printf("Ошибка при получении номеров заказов: %v", err)
			continue
		}

		for _, orderNumber := range orderNumbers {
			orderChan <- orderNumber
		}
	}
}

func (s *Store) Send(result json2.OrderResponse) error {
	query := `UPDATE order_history SET status  = $1, accrual = $2 WHERE  = order_number = $3`

	tx, err := s.db.Begin()
	go func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query, result.Status, result.Accrual, result.Order)
	if err != nil {
		return err
	}
	query1 := `UPDATE loyalty_accounts SET current = current + $1 WHERE order_number = $2`
	_, err = tx.Exec(query1, result.Accrual, result.Order)
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
		log.Printf("error in checking for database presence: %v", err)
		return
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE users (
        user_id SERIAL PRIMARY KEY,
        username VARCHAR(255) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL
    );`)
		if err != nil {
			log.Printf("database creation error: %v", err)
		}
	}
	exists, err = tableExists(db, "orders")
	if err != nil {
		log.Fatal("dB error")
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    order_number VARCHAR(255) UNIQUE NOT NULL,
     FOREIGN KEY (username) REFERENCES users(username)
);
`)
		if err != nil {
			log.Fatal("dB error")
		}
	}
	exists, err = tableExists(db, "order_history")
	if err != nil {
		log.Fatal("dB error")
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE order_history (
			history_id SERIAL PRIMARY KEY,
			order_number VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			status VARCHAR(20) NOT NULL,
			accrual INTEGER  NOT NULL,
			withdrawals INTEGER DEFAULT 0,
			processed_at TIMESTAMP,
			FOREIGN KEY (order_number) REFERENCES orders(order_number),
			FOREIGN KEY (username) REFERENCES users(username)
		);`)
		if err != nil {
			log.Fatal("dB error")
		}
	}
	exists, err = tableExists(db, "loyalty_accounts")
	if err != nil {
		log.Fatal("dB error")
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE loyalty_accounts (
    account_id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    current INTEGER DEFAULT 0,
	withdrawn INTEGER DEFAULT 0,
    FOREIGN KEY (username) REFERENCES users(username)
);`)
		if err != nil {
			log.Fatal("dB error")
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
