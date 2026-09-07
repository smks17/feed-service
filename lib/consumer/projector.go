package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/smks17/feed-service/lib/events"
)

type ErrUnprocessable struct{ err error }

func (e ErrUnprocessable) Error() string { return e.err.Error() }
func (e ErrUnprocessable) Unwrap() error { return e.err }

func except(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "22") {
		return ErrUnprocessable{err}
	}
	return err
}

type Projector struct {
	db *pgxpool.Pool
}

func NewProjector(db *pgxpool.Pool) *Projector {
	return &Projector{db: db}
}

func (p *Projector) Apply(ctx context.Context, raw []byte) error {
	var env events.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return ErrUnprocessable{fmt.Errorf("decode envelope: %w", err)}
	}
	if env.EventID == "" {
		return ErrUnprocessable{errors.New("envelope has no event_id")}
	}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var seen bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id) VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING true
	`, env.EventID).Scan(&seen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // already applied
	}
	if err != nil {
		return except(fmt.Errorf("record event: %w", err))
	}

	if err := p.project(ctx, tx, env); err != nil {
		return except(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (p *Projector) project(ctx context.Context, tx pgx.Tx, env events.Envelope) error {
	switch env.EventType {
	case events.PostCreated:
		var d events.PostData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO posts (id, author_id, content, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, d.ID, d.AuthorID, d.Content, d.CreatedAt)
		return err

	case events.PostDestroyed:
		var d events.PostData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx, `DELETE FROM posts WHERE id = $1`, d.ID)
		return err

	case events.Liked:
		var d events.LikeData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO likes (post_id, user_id) VALUES ($1, $2)
			ON CONFLICT (post_id, user_id) DO NOTHING
		`, d.PostID, d.UserID)
		return err

	case events.Unliked:
		var d events.LikeData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx,
			`DELETE FROM likes WHERE post_id = $1 AND user_id = $2`, d.PostID, d.UserID)
		return err

	case events.Followed:
		var d events.FollowData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO follows (follower_id, following_id) VALUES ($1, $2)
			ON CONFLICT (follower_id, following_id) DO NOTHING
		`, d.FollowerID, d.FollowingID)
		return err

	case events.Unfollowed:
		var d events.FollowData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return ErrUnprocessable{err}
		}
		_, err := tx.Exec(ctx,
			`DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`,
			d.FollowerID, d.FollowingID)
		return err

	case events.Commented:
		return nil

	default:
		return ErrUnprocessable{fmt.Errorf("unknown event type %q", env.EventType)}
	}
}
