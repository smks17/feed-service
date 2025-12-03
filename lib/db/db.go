package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

func Connect(ctx context.Context, addr string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), addr)
	if err != nil {
		log.Println("Unable to connect to database: ", err)
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		log.Fatal("Can not ping database", err)
		return nil, err
	}
	log.Print("Connect to database")

	return conn, nil
}
