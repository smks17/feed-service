package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/smks17/feed-service/lib/feed"
)

const ExpirationExploreFeedCache = time.Minute

type ExploreFeedCache struct {
	rdb *redis.Client
}

func (hfc *ExploreFeedCache) Set(ctx context.Context, userId uint32, post []feed.Post) error {
	cache_key := fmt.Sprint("explore-user-", userId)
	json, err := json.Marshal(post)
	if err != nil {
		return err
	}
	return hfc.rdb.SetEX(ctx, cache_key, json, ExpirationExploreFeedCache).Err()
}

func (hfc *ExploreFeedCache) Get(ctx context.Context, userId uint32) ([]feed.Post, error) {
	cache_key := fmt.Sprint("explore-user-", userId)

	data, err := hfc.rdb.Get(ctx, cache_key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var post []feed.Post
	if data != "" {
		err := json.Unmarshal([]byte(data), &post)
		if err != nil {
			return nil, err
		}
	}
	return post, nil
}

func (hfc *ExploreFeedCache) Delete(ctx context.Context, userId uint32) {
	cache_key := fmt.Sprint("explore-user-", userId)
	hfc.rdb.Del(ctx, cache_key)
}
