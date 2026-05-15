package db

import (
	"context"
	"github.com/jackc/pgx/v5"
	"os"
)

func Connect() (*pgx.Conn, error) {
	// Format: postgres://username:password@localhost:5432/database_name
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	return conn, err
}
