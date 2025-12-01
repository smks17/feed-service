package cache

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/smks17/feed-service/lib/feed"
)

type FeedCache struct {
	HomeFeed interface {
		Set(ctx context.Context, userId uint32, feed []feed.Post) error
		Get(ctx context.Context, userId uint32) ([]feed.Post, error)
		Delete(ctx context.Context, userId uint32)
	}
}

func NewFeedCache(rdb *redis.Client) FeedCache {
	return FeedCache{
		HomeFeed: &HomeFeedCache{rdb: rdb},
	}
}
