package events

import (
	"encoding/json"
	"time"
)

type Type string

const (
	PostCreated   Type = "create_post"
	PostDestroyed Type = "destroy_post"
	Liked         Type = "like"
	Unliked       Type = "unlike"
	Commented     Type = "comment"
	Followed      Type = "follow"
	Unfollowed    Type = "unfollow"
)

type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  Type            `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Key        string          `json:"key"`
	Data       json.RawMessage `json:"data"`
}

type PostData struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type LikeData struct {
	PostID int64 `json:"post_id"`
	UserID int64 `json:"user_id"`
}

type FollowData struct {
	FollowerID  int64 `json:"follower_id"`
	FollowingID int64 `json:"following_id"`
}
