package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func Connect(driverName string, addr string) (*sql.DB, error) {
	db, err := sql.Open(driverName, addr)
	if err != nil {
		log.Fatal("Can not connect to database", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Can not ping database", err)
		return nil, err
	}
	log.Print("Connect to database")

	return db, nil
}
