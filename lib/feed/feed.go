package feed

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Feed struct {
	Posts interface {
		GetHomeFeed(ctx context.Context, userId uint32) ([]Post, error)
	}
}

func NewFeed(db *pgx.Conn) Feed {
	return Feed{
		Posts: &PostStore{db},
	}
}
