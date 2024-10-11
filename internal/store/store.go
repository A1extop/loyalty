package store

import (
	"database/sql"
	"log"
	"time"

	"fmt"

	"net/http"

	"github.com/A1extop/loyalty/internal/hash"
	json2 "github.com/A1extop/loyalty/internal/json"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Store struct {
	db *sql.DB
}
type Storage interface {
	AddUsers(login string, key string, password string, c *gin.Context)
	CheckAvailability(login string, password string, c *gin.Context) bool
	SendingData(login string, number string, c *gin.Context)
	CheckNumber(c *gin.Context, number string) bool
	CheckUserOrders(c *gin.Context, login string, num string) bool

	Orders(login string, c *gin.Context) ([]json2.History, error)
}

func (s *Store) SendingData(login string, number string, c *gin.Context) {
	query := `INSERT INTO orders (username, order_number) VALUES ($1, $2)`
	_, err := s.db.Exec(query, login, number)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusAccepted, "The new order number has been accepted for processing;")
}

func (s *Store) Orders(login string, c *gin.Context) ([]json2.History, error) {
	slHistory := make([]json2.History, 0)
	query := `SELECT history_id, order_id, user_id, status, accrual, uploaded_at 
              FROM order_history WHERE user_id = $1 ORDER BY uploaded_at DESC`
	rows, err := s.db.Query(query, login)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var historyID int
	var orderId int
	var userId int
	var status string
	var accrual int
	var timeStamp time.Time

	for rows.Next() {
		if err := rows.Scan(&historyID, &orderId, &userId, &status, &accrual, &timeStamp); err != nil {
			continue
		}
		history := json2.History{
			HistoryID: historyID,
			OrderId:   orderId,
			UserId:    userId,
			Status:    status,
			Accrual:   accrual,
			Uploaded:  timeStamp,
		}
		slHistory = append(slHistory, history)
	}
	return slHistory, nil
}

func (s *Store) CheckUserOrders(c *gin.Context, login string, num string) bool {
	query := `SELECT EXISTS(SELECT 1 FROM orders WHERE username = $1 AND order_number = $2)`

	var exists bool
	err := s.db.QueryRow(query, login, num).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return true
	}
	if exists {
		c.String(http.StatusOK, "Everything is fine")
	}
	return exists
}

func (s *Store) CheckNumber(c *gin.Context, number string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM orders WHERE order_number=$1)"
	err := s.db.QueryRow(query, number).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return false
	}
	if exists {
		c.String(http.StatusConflict, "Conflict")
		return false
	}
	return true
}

func (s *Store) AddUsers(login string, key string, password string, c *gin.Context) {

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.QueryRow(query, login).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if exists {
		c.String(http.StatusConflict, "this login is already taken")
		return
	}
	hashedPassword, err := hash.HashPassword(password, key) // подумаю как протянуть ещё
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	insertQuery := "INSERT INTO users (username, password_hash) VALUES ($1, $2)"
	_, err = s.db.Exec(insertQuery, login, hashedPassword)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "user successfully registered")
}

func (s *Store) CheckAvailability(login string, password string, c *gin.Context) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)"
	err := s.db.QueryRow(query, login).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed 1")
		return false
	}
	if !exists {
		c.String(http.StatusUnauthorized, "incorrect login/password pair")
		return false
	}
	hashedPassword, err := hash.HashPassword(password, "secretKey") // подумаю как протянуть ещё
	if err != nil {
		c.String(http.StatusInternalServerError, "failed 2")
		return false
	}
	query1 := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND password_hash =$2)"
	err = s.db.QueryRow(query1, login, hashedPassword).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed 3")
		return false
	}
	if !exists {
		c.String(http.StatusUnauthorized, "incorrect login/password pair")
		return false
	}
	return true
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
			order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL,
			accrual INTEGER  NOT NULL,
			FOREIGN KEY (order_id) REFERENCES orders(order_id),
			FOREIGN KEY (user_id) REFERENCES users(user_id)
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
