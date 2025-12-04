package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/smks17/feed-service/lib/feed"
)

const (
	ExpirationPopularFeedCache = 24 * time.Hour // But it will be updated frequently
	CACHE_KEY_POPULAR_FEED     = "popular-feed"
)

type PopularFeedCache struct {
	rdb *redis.Client
}

func (hfc *PopularFeedCache) Set(ctx context.Context, post []feed.Post) error {
	json, err := json.Marshal(post)
	if err != nil {
		return err
	}
	// TODO: set just ids
	return hfc.rdb.SetEX(ctx, CACHE_KEY_POPULAR_FEED, json, ExpirationPopularFeedCache).Err()
}

func (hfc *PopularFeedCache) Get(ctx context.Context) ([]feed.Post, error) {
	data, err := hfc.rdb.Get(ctx, CACHE_KEY_POPULAR_FEED).Result()
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

func (hfc *PopularFeedCache) Delete(ctx context.Context) {
	hfc.rdb.Del(ctx, CACHE_KEY_POPULAR_FEED)
}
