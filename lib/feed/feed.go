package feed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type feedCacheType func(context.Context) ([]Post, error)

type Feed struct {
	Posts interface {
		GetHomeFeed(ctx context.Context, userId uint32) ([]Post, error)
		GetExploreFeed(ctx context.Context, userId uint32, getPopularFeedFromCache feedCacheType) ([]Post, error)
		GetRandomFeed(ctx context.Context, limit int) ([]Post, error)
		GetPopularFeed(ctx context.Context) ([]Post, error)
	}
}

func NewFeed(db *pgxpool.Pool) Feed {
	return Feed{
		Posts: &PostStore{db},
	}
}
