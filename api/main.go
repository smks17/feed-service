package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/smks17/feed-service/lib/cache"
	"github.com/smks17/feed-service/lib/db"
	"github.com/smks17/feed-service/lib/feed"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file: %s", err)
	}
	config := setConfig()

	ctx := context.Background()

	db, err := db.Connect(ctx, config.db.addr)
	if err != nil {
		os.Exit(1)
	}
	defer db.Close(ctx)

	rdb := cache.NewRedisClient(
		fmt.Sprintf("%s:%d", config.redis.addr, config.redis.port),
		config.redis.password,
		config.redis.db,
	)
	defer rdb.Close()
	status, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalln("Redis connection was refused: ", err)
	}
	fmt.Println("Stats of redis: ", status)

	feed := feed.NewFeed(db)

	feedCache := cache.NewFeedCache(rdb)

	app := newApp(ctx, &feed, config, &feedCache)
	router := app.mount()
	log.Fatal(app.run(router))
}
