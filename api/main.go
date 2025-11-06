package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/smks17/feed_service/lib/db"
	"github.com/smks17/feed_service/lib/feed"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}
	config := setConfig()

	db, err := db.Connect("sqlite3", "/home/mahdi/Documents/code/switter/db.sqlite3")
	if err != nil {
		return
	}
	defer db.Close()

	feed := feed.NewFeed(db)

	app := newApp(&feed, config)
	router := app.mount()
	log.Fatal(app.run(router))
}
