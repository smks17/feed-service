package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/smks17/feed-service/lib/cache"
	"github.com/smks17/feed-service/lib/consumer"
	"github.com/smks17/feed-service/lib/db"
	"github.com/smks17/feed-service/lib/env"
	"github.com/smks17/feed-service/lib/feed"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
	}
	config := setConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := db.Migrate(config.db.addr); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	db, err := db.Connect(ctx, config.db.addr)
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

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
	go app.UpdatePopularFeedCache(app.ctx) // update popular feed cache in background
	go runConsumer(ctx, db)
	log.Fatal(app.run(router))
}

func runConsumer(ctx context.Context, pool *pgxpool.Pool) {
	brokers, err := env.CheckEnv("KAFKA_URL")
	if err != nil {
		log.Fatal("KAFKA_URL is not set")
	}
	postTopic, err := env.CheckEnv("KAFKA_POST_TOPIC")
	if err != nil {
		log.Fatal("KAFKA_POST_TOPIC is not set")
	}
	interactionTopic, err := env.CheckEnv("KAFKA_INTERACTION_TOPIC")
	if err != nil {
		log.Fatal("KAFKA_INTERACTION_TOPIC is not set")
	}

	c := consumer.New(
		strings.Split(brokers, ","),
		[]string{postTopic, interactionTopic},
		env.GetEnv("FEED_CONSUMER_GROUP", "feed-service"),
		consumer.NewProjector(pool),
	)
	defer c.Close()

	log.Printf("Consuming %s and %s", postTopic, interactionTopic)
	if err := c.Run(ctx); err != nil {
		log.Printf("Consumer stopped: %v", err)
	}
}
