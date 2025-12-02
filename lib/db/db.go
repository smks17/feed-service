package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func Connect(ctx context.Context, host string, addr string, dbName string) (*pgx.Conn, error) {
	url := fmt.Sprintf("%s://%s/%s", host, addr, dbName)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		log.Println("Unable to connect to database: %v\n", err)
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		log.Fatal("Can not ping database", err)
		return nil, err
	}
	log.Print("Connect to database")

	return conn, nil
}
