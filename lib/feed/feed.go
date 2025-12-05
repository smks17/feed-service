package feed

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Feed struct {
	Posts interface {
		GetHomeFeed(ctx context.Context, userId uint32) ([]Post, error)
		GetExploreFeed(ctx context.Context, userId uint32) ([]Post, error)
		GetRandomFeed(ctx context.Context) ([]Post, error)
		GetPopularFeed(ctx context.Context) ([]Post, error)
	}
}

func NewFeed(db *pgx.Conn) Feed {
	return Feed{
		Posts: &PostStore{db},
	}
}
