package feed

import (
	"database/sql"
)

type Feed struct {
	Posts interface {
		GetHomeFeed(userId uint32) ([]Post, error)
	}
}

func NewFeed(db *sql.DB) Feed {
	return Feed{
		Posts: &PostStore{db},
	}
}
