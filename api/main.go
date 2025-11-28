package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/smks17/feed_service/lib/cache"
	"github.com/smks17/feed_service/lib/db"
	"github.com/smks17/feed_service/lib/feed"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}
	config := setConfig()

	ctx := context.Background()

	db, err := db.Connect("sqlite3", "/home/mahdi/Documents/code/switter/db.sqlite3")
	if err != nil {
		return
	}
	defer db.Close()

	rdb := cache.NewRedisClient("localhost:6379", "", 0)
	defer rdb.Close()
	status, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalln("Redis connection was refused")
	}
	fmt.Println("Stats of redis: ", status)

	feed := feed.NewFeed(db)

	feedCache := cache.NewFeedCache(rdb)

	app := newApp(ctx, &feed, config, &feedCache)
	router := app.mount()
	log.Fatal(app.run(router))
}
