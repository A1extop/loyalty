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
	//Отправление заказа
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
	query := `INSERT INTO order_history  (username, order_number) VALUES ($1, $2)` /////////////////////////////////////
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
	var status string
	var accrual int
	var timeStamp time.Time

	for rows.Next() {
		if err := rows.Scan(&order, &status, &accrual, &withdrawals, &timeStamp); err != nil {
			continue
		}
		accFloat := float64(accrual) / 100
		withdrawFloat := float64(withdrawals) / 100

		history := json2.History{
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
	query := `SELECT EXISTS(SELECT 1 FROM order_history  WHERE username = $1 AND order_number = $2)` ////////////////////////////////  CheckUserOrders и CheckNumber как будто можно совместить

	var exists bool
	err := s.db.QueryRow(query, login, num).Scan(&exists)
	if err != nil {
		return false, errors.Join(err, domain.ErrInternal)
	}
	return exists, nil
}

func (s *Store) CheckNumber(number string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM order_history  WHERE order_number=$1)" /////////////////////////////////////
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
	query := `SELECT current FROM loyalty_accounts WHERE username = $1`
	err = s.db.QueryRow(query, login).Scan(&balance)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	balanceFloat := float64(balance)
	balanceFloat /= 100

	if balanceFloat-num < 0 {
		return errors.Join(errors.New("there are insufficient funds in the account"), domain.ErrPaymentRequired)
	}

	query = `SELECT withdrawals FROM order_history WHERE username = $1 AND order_number = $2`
	var withdrawals int
	row := tx.QueryRow(query, login, order)
	err = row.Scan(&withdrawals)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)

	}
	currentBalance := balanceFloat - num
	withdrawalsFloat := float64(withdrawals)

	if withdrawalsFloat != 0.0 {
		return errors.Join(errors.New("there has already been a write-off for this order"), domain.ErrUnprocessableEntity) //проверка здесь происходит на то, не списывались ли в счёт этого заказа уже баллы, возвращаю 422, но не уверен
	}

	_, err = tx.Exec("UPDATE order_history SET withdrawals  = $1 WHERE username = $2 AND order_number = $3", num*100, login, order)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	_, err = tx.Exec("UPDATE loyalty_accounts SET current = $1 WHERE username = $2", currentBalance*100, login)
	if err != nil {
		return errors.Join(err, domain.ErrInternal)
	}
	_, err = tx.Exec("UPDATE loyalty_accounts SET withdrawn = withdrawn + $1 WHERE username = $2", num*100, login)
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
	query := `SELECT order_number 
FROM order_history 
WHERE status IN ('REGISTERED', 'PROCESSING') 
   OR status IS NULL`
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
		log.Printf("error in checking for database presence: %v", err)
		return
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE users (
    username VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL
);`)
		if err != nil {
			log.Printf("database creation error: %v", err)
		}
	}

	exists, err = tableExists(db, "order_history")
	if err != nil {
		log.Fatal("dB error 1:", err)
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE order_history (
			order_number VARCHAR(255) NOT NULL,
			username VARCHAR(255) NOT NULL,
			status VARCHAR(20),
			accrual INTEGER  DEFAULT 0,
			withdrawals INTEGER DEFAULT 0,
			processed_at TIMESTAMP,
			FOREIGN KEY (username) REFERENCES users(username)
		);`)
		if err != nil {
			log.Fatal("dB error 2:", err)
		}
	}
	exists, err = tableExists(db, "loyalty_accounts")
	if err != nil {
		log.Fatal("dB error")
	}
	if !exists {
		_, err = db.Exec(`CREATE TABLE loyalty_accounts (
    username VARCHAR(255) PRIMARY KEY,
    current INTEGER DEFAULT 0,
	withdrawn INTEGER DEFAULT 0,
    FOREIGN KEY (username) REFERENCES users(username)
);`)
		if err != nil {
			log.Fatal("dB error 3:", err)
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
