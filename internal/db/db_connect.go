package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
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

	return &Database{
		Pool: pool,
	}, nil
}

func (db *Database) Close() {
	db.Pool.Close()
}
