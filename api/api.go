package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/smks17/feed_service/lib/cache"
	"github.com/smks17/feed_service/lib/env"
	"github.com/smks17/feed_service/lib/feed"

	"github.com/go-chi/chi/v5"
)

type APPConfig struct {
	addr string
	db   DBConfig
}

type DBConfig struct {
	addr string
}

func setConfig() APPConfig {
	return APPConfig{
		addr: env.GetEnv("ADDR", "127.0.0.1:8080"),
		db:   DBConfig{addr: env.GetEnv("DB_PATH", "/db.sqlite3")},
	}
}

type APP struct {
	ctx       context.Context
	config    APPConfig
	feed      *feed.Feed
	feedCache *cache.FeedCache
	// httServer *http.Server
}

func newApp(ctx context.Context, feed *feed.Feed, config APPConfig, feedCache *cache.FeedCache) *APP {
	return &APP{ctx: ctx, config: config, feed: feed, feedCache: feedCache}
}

func (app *APP) mount() *chi.Mux {
	route := chi.NewRouter()

	route.Route("/feed", func(r chi.Router) {
		r.Route("/home", func(r chi.Router) {
			r.Get("/{userID}", app.getHomePostHandler)
		})
	})

	return route
}

func (app *APP) run(r *chi.Mux) error {
	server := &http.Server{
		Addr:         app.config.addr,
		Handler:      r,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	err := server.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}

const MAX_SIZE_READ int64 = 1_048_578

func readJSON(writer http.ResponseWriter, reader *http.Request, data any) error {
	http.MaxBytesReader(writer, reader.Body, MAX_SIZE_READ)
	decoder := json.NewDecoder(reader.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(data)
}

func (app *APP) jsonResponse(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

type GetPostPayload struct {
	UserId  uint32   `json:"user"`
	PostIds []uint32 `json:"ids"`
}

func (app *APP) getHomePostHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		log.Fatal(err)
		return
	}

	var posts []feed.Post

	cachePosts, err := app.feedCache.HomeFeed.Get(app.ctx, uint32(userID))
	if err != nil {
		log.Fatal("Error in get feeds of user %d from cache: %v", userID, err)
		return
	} else if cachePosts != nil {
		posts = cachePosts
		log.Println("Hit cache for for user ", userID)
	} else {
		dbPosts, err := app.feed.Posts.GetHomeFeed(uint32(userID))
		if err != nil {
			log.Fatal(err)
			return
		}
		err = app.feedCache.HomeFeed.Set(app.ctx, uint32(userID), dbPosts)
		if err != nil {
			log.Fatal("Error in set feeds of user %d from cache: %v", userID, err)
			return
		}
		log.Println("Set from cache for user ", userID)
		posts = dbPosts
	}

	postIDs := make([]uint32, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}
	ret := GetPostPayload{UserId: uint32(userID), PostIds: postIDs}
	app.jsonResponse(w, 200, ret)
}
