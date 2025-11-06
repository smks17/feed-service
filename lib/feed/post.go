package feed

import (
	"database/sql"
	"log"
	"time"
)

type Post struct {
	ID        uint32
	Content   string
	CreatedAt time.Time
	Author    uint32
}

type PostStore struct {
	db *sql.DB
}

func (ps *PostStore) GetHomeFeed(userId uint32) ([]Post, error) {
	query := `
        SELECT p.id, p.content, p.created_at, p.author_id
        FROM posts_post p
        JOIN interactions_followlinks f
          ON f.following_id = p.author_id
        WHERE f.follower_id = ?
        ORDER BY p.created_at DESC;
    `

	rows, err := ps.db.Query(query, userId, userId)
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
