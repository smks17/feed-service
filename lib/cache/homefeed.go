package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/smks17/feed-service/lib/feed"
)

const ExpirationHomeFeedCache = time.Minute

type HomeFeedCache struct {
	rdb *redis.Client
}

func (hfc *HomeFeedCache) Set(ctx context.Context, userId uint32, post []feed.Post) error {
	cache_key := fmt.Sprint("homefeed-user-", userId)
	json, err := json.Marshal(post)
	if err != nil {
		return err
	}
	return hfc.rdb.SetEX(ctx, cache_key, json, ExpirationHomeFeedCache).Err()
}

func (hfc *HomeFeedCache) Get(ctx context.Context, userId uint32) ([]feed.Post, error) {
	cache_key := fmt.Sprint("homefeed-user-", userId)

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

func (hfc *HomeFeedCache) Delete(ctx context.Context, userId uint32) {
	cache_key := fmt.Sprint("homefeed-user-", userId)
	hfc.rdb.Del(ctx, cache_key)
}
