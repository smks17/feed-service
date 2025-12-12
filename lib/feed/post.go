package feed

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	ID        uint32
	Content   string
	CreatedAt time.Time
	Author    uint32
}

type PostStore struct {
	db *pgxpool.Pool
}

func (ps *PostStore) GetHomeFeed(ctx context.Context, userId uint32) ([]Post, error) {
	query := `
        SELECT p.id, p.content, p.created_at, p.author_id
        FROM posts_post p
        JOIN interactions_followlinks f
          ON f.following_id = p.author_id
        WHERE f.follower_id = $1 AND p.created_at >= NOW() - INTERVAL '3 days'
        ORDER BY p.created_at DESC;
    `

	rows, err := ps.db.Query(ctx, query, userId)
	if err != nil {
		log.Printf("Error in get feeds of user %d: %v", userId, err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Content, &p.CreatedAt, &p.Author); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (ps *PostStore) GetPopularFeed(ctx context.Context) ([]Post, error) {
	query := `
		SELECT p.id, p.content, p.created_at, p.author_id
		FROM posts_post AS p
		LEFT JOIN interactions_like AS l
			ON l.post_id = p.id
		WHERE p.created_at >= NOW() - INTERVAL '3 days'
		GROUP BY p.id, p.content, p.created_at, p.author_id
		ORDER BY COUNT(l.id) DESC LIMIT 20;
	`

	rows, err := ps.db.Query(ctx, query)
	if err != nil {
		log.Printf("Error in get popular feed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Content, &p.CreatedAt, &p.Author); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (ps *PostStore) GetRandomFeed(ctx context.Context, limit int) ([]Post, error) {
	query := `
		SELECT * FROM (
			SELECT p.id, p.content, p.created_at, p.author_id
			FROM posts_post AS p
			WHERE p.created_at >= NOW() - INTERVAL '3 days'
			ORDER BY random()
			LIMIT $1
		) AS sub
		ORDER BY created_at DESC;
	`

	rows, err := ps.db.Query(ctx, query, limit)
	if err != nil {
		log.Printf("Error in get random feed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Content, &p.CreatedAt, &p.Author); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func pickRandom[T any](list []T, n int) []T {
	shuffled := append([]T(nil), list...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	if len(shuffled) > n {
		return shuffled[:n]
	}
	return shuffled
}

type postsResult struct {
	posts []Post
	err   error
}

func (ps *PostStore) GetExploreFeed(ctx context.Context, userId uint32, getPopularFeedFromCache feedCacheType) ([]Post, error) {
	var posts []Post

	localCtx := context.Context(ctx)

	homeCh := make(chan postsResult, 1)
	popularCh := make(chan postsResult, 1)
	randomCh := make(chan postsResult, 1)

	go func() {
		p, err := ps.GetHomeFeed(localCtx, userId)
		if err == nil {
			p = pickRandom(p, 5)
		}
		homeCh <- postsResult{posts: p, err: err}
	}()

	go func() {
		p, err := getPopularFeedFromCache(localCtx)
		if err == nil {
			p = pickRandom(p, 10)
		}
		popularCh <- postsResult{posts: p, err: err}
	}()

	go func() {
		p, err := ps.GetRandomFeed(localCtx, 20)
		randomCh <- postsResult{posts: p, err: err}
	}()

	// collect
	var (
		homeRes    postsResult
		popularRes postsResult
		randomRes  postsResult
	)

	// wait for all three (with ctx cancel if you want early cancel)
	for i := 0; i < 3; i++ {
		select {
		case r := <-homeCh:
			homeRes = r
		case r := <-popularCh:
			popularRes = r
		case r := <-randomCh:
			randomRes = r
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	// aggregate errors (only non-nil ones)
	var errs []error
	if homeRes.err != nil {
		errs = append(errs, homeRes.err)
	}
	if popularRes.err != nil {
		errs = append(errs, popularRes.err)
	}
	if randomRes.err != nil {
		errs = append(errs, randomRes.err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	// build final list
	posts = append(posts, homeRes.posts...)
	posts = append(posts, popularRes.posts...)
	posts = append(posts, randomRes.posts...)

	return posts, nil
}
