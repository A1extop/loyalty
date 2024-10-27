package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func tableExists(ctx context.Context, pool *pgxpool.Pool, tableName string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '%s');", tableName)
	err := pool.QueryRow(ctx, query).Scan(&exists)
	return exists, err
}
func NewDatabase(ctx context.Context, dsn string) (*Database, error) {
	//dataSourceName := fmt.Sprintf("host=%s port=%s user=%s database=%s password=%s sslmode=%s",
	//	cfg.Pg.Host, cfg.Pg.Port, cfg.Pg.Username, cfg.Pg.Database, cfg.Pg.Password, cfg.Pg.Ssl)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	exists, err := tableExists(ctx, pool, "users")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = pool.Exec(ctx, `CREATE TABLE users (
    username VARCHAR(255) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL
);`)
		if err != nil {
			log.Fatal("database creation error1:", err)
		}
	}

	exists, err = tableExists(ctx, pool, "order_history")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = pool.Exec(ctx, `CREATE TABLE order_history (
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
	exists, err = tableExists(ctx, pool, "loyalty_accounts")
	if err != nil {
		log.Fatal("error in checking for database presence: ", err)
	}
	if !exists {
		_, err = pool.Exec(ctx, `CREATE TABLE loyalty_accounts (
    username VARCHAR(255) PRIMARY KEY,
    current INTEGER DEFAULT 0,
	withdrawn INTEGER DEFAULT 0,
    FOREIGN KEY (username) REFERENCES users(username)
);`)
		if err != nil {
			log.Fatal("database creation error3:", err)
		}
	}

	return &Database{
		Pool: pool,
	}, nil
}

func (db *Database) Close() {
	db.Pool.Close()
}
