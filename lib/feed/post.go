package feed

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type Post struct {
	ID        uint32
	Content   string
	CreatedAt time.Time
	Author    uint32
}

type PostStore struct {
	db *pgx.Conn
}

func (ps *PostStore) GetHomeFeed(ctx context.Context, userId uint32) ([]Post, error) {
	query := `
        SELECT p.id, p.content, p.created_at, p.author_id
        FROM posts_post p
        JOIN interactions_followlinks f
          ON f.following_id = p.author_id
        WHERE f.follower_id = ? AND p.created_at >= NOW() - INTERVAL '3 days'
        ORDER BY p.created_at DESC;
    `

	rows, err := ps.db.Query(ctx, query, userId, userId)
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

func (ps *PostStore) GetRandomFeed(ctx context.Context) ([]Post, error) {
	query := `
		SELECT p.id, p.content, p.created_at, p.author_id
		FROM posts_post AS p
		WHERE p.created_at >= NOW() - INTERVAL '3 days'
		ORDER BY random()
		ORDER BY p.created_at DESC LIMIT 20;
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

func pickRandom[T any](list []T, n int) []T {
	shuffled := make([]T, len(list))
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:n]
}

func (ps *PostStore) GetExploreFeed(ctx context.Context, userId uint32) ([]Post, error) {
	var posts []Post

	var homePosts, popularPosts, randomPosts []Post
	var err1, err2, err3 error

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		homePosts, err1 = ps.GetHomeFeed(ctx, userId)
		homePosts = pickRandom(homePosts, 5)
	}()

	go func() {
		defer wg.Done()
		popularPosts, err2 = ps.GetPopularFeed(ctx)
		homePosts = pickRandom(homePosts, 10)
	}()

	go func() {
		defer wg.Done()
		randomPosts, err3 = ps.GetRandomFeed(ctx)
		homePosts = pickRandom(homePosts, 20)
	}()

	wg.Wait()

	if err1 != nil || err2 != nil || err3 != nil {
		return nil, errors.Join(err1, err2, err3)
	}

	posts = append(posts, homePosts...)
	posts = append(posts, popularPosts...)
	posts = append(posts, randomPosts...)

	return posts, nil
}
