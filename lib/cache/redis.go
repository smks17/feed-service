package cache

import "github.com/go-redis/redis/v8"

func NewRedisClient(addr, password string, db int) *redis.Client {
	opts := &redis.Options{
		Addr: addr,
		DB:   db,
	}

	if password != "" {
		opts.Password = password
	}

	return redis.NewClient(opts)
}
