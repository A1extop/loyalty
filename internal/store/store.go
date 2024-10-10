package store

import (
	"database/sql"
	"log"

	"fmt"

	"net/http"

	"github.com/A1extop/loyalty/internal/hash"
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
	CheckUserOrders(c *gin.Context, login string) bool
}

func (s *Store) SendingData(login string, number string, c *gin.Context) {
	query := `INSERT INTO orders (user_login, order_number) VALUES ($1, $2)`
	_, err := s.db.Exec(query, login, number)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
}

func (s *Store) CheckUserOrders(c *gin.Context, login string) bool {
	query := `SELECT EXISTS(SELECT 1 FROM orders WHERE user_login = $1)`

	var exists bool
	err := s.db.QueryRow(query, login).Scan(&exists)
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
		c.String(http.StatusInternalServerError, "failed")
		return false
	}
	if !exists {
		c.String(http.StatusUnauthorized, "incorrect login/password pair")
		return false
	}
	hashedPassword, err := hash.HashPassword(password, "secretKey") // подумаю как протянуть ещё
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return false
	}
	query1 := "SELECT EXISTS(SELECT 1 FROM users WHERE username=$1 AND password=$2)"
	err = s.db.QueryRow(query1, login, hashedPassword).Scan(&exists)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
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
	if !exists {
		_, err = db.Exec(`CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    order_number VARCHAR(255) UNIQUE NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(user_id)
	);`)
	}
}

func ConnectDB(connectionToBD string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionToBD)
	if err != nil {
		return nil, err
	}
	return db, nil
}
