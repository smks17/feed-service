package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/smks17/feed-service/lib/cache"
	"github.com/smks17/feed-service/lib/env"
	"github.com/smks17/feed-service/lib/feed"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type APPConfig struct {
	addr  string
	db    DBConfig
	redis RedisConfig
}

type DBConfig struct {
	addr string
}

type RedisConfig struct {
	addr     string
	port     int
	password string
	db       int
}

func setConfig() APPConfig {
	return APPConfig{
		addr: env.GetEnv("ADDR", "127.0.0.1:8080"),
		db: DBConfig{
			addr: env.GetEnv("POSTGRES_URL", "127.0.0.1:5432"),
		},
		redis: RedisConfig{
			addr:     env.GetEnv("REDIS_HOSTNAME", "127.0.0.1"),
			port:     env.GetInt("REDIS_PORT", 6379),
			password: env.GetEnv("REDIS_PASSWORD", ""),
			db:       env.GetInt("FEED_SERVICE_REDIS_DB", 0),
		},
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

	route.Use(middleware.Logger)
	route.Use(middleware.Recoverer)
	route.Use(middleware.Timeout(10 * time.Second))

	route.Route("/feed", func(r chi.Router) {
		r.Route("/home", func(r chi.Router) {
			r.Get("/{userID}", app.getHomePostHandler)
		})
		r.Route("/explore", func(r chi.Router) {
			r.Get("/{userID}", app.getExplorePostHandler)
		})
		r.Get("/popular", app.getPopularPostHandler)
		r.Get("/random", app.getRandomPostHandler)
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

func (app *APP) UpdatePopularFeedCache() {
	if err := app.updatePopularFeed(); err != nil {
		log.Println("Error updating popular feed:", err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := app.updatePopularFeed(); err != nil {
				log.Println("Error updating popular feed:", err)
			}
		case <-app.ctx.Done():
			fmt.Println("Stop signal received, exiting goroutine.")
			return
		}
	}
}

func (app *APP) updatePopularFeed() error {
	dbPosts, err := app.feed.Posts.GetPopularFeed(app.ctx)
	if err != nil {
		return err
	}
	err = app.feedCache.PopularFeed.Set(app.ctx, dbPosts)
	if err != nil {
		return err
	}
	fmt.Println("Set popular feed to cache")
	return nil
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
		dbPosts, err := app.feed.Posts.GetHomeFeed(app.ctx, uint32(userID))
		if err != nil {
			log.Fatal(err)
			return
		}
		// set cache in background
		go func() {
			err = app.feedCache.HomeFeed.Set(app.ctx, uint32(userID), dbPosts)
			if err != nil {
				log.Fatal("Error in set feeds of user %d from cache: %v", userID, err)
				return
			}
			log.Println("Set from cache for user ", userID)
		}()
		posts = dbPosts
	}

	postIDs := make([]uint32, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}
	ret := GetPostPayload{UserId: uint32(userID), PostIds: postIDs}
	app.jsonResponse(w, 200, ret)
}

func (app *APP) getExplorePostHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		log.Fatal(err)
		return
	}

	var posts []feed.Post

	cachePosts, err := app.feedCache.ExploreFeed.Get(app.ctx, uint32(userID))
	if err != nil {
		log.Fatal("Error in get feeds of user %d from cache: %v", userID, err)
		return
	} else if cachePosts != nil {
		posts = cachePosts
		log.Println("Hit cache for for user ", userID)
	} else {
		dbPosts, err := app.feed.Posts.GetExploreFeed(app.ctx, uint32(userID), app.feedCache.PopularFeed.Get)
		if err != nil {
			log.Fatal(err)
			return
		}
		// set cache in background
		go func() {
			err = app.feedCache.ExploreFeed.Set(app.ctx, uint32(userID), dbPosts)
			if err != nil {
				log.Fatal("Error in set feeds of user %d from cache: %v", userID, err)
				return
			}
			log.Println("Set from cache for user ", userID)
		}()
		posts = dbPosts
	}

	postIDs := make([]uint32, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}
	ret := GetPostPayload{UserId: uint32(userID), PostIds: postIDs}
	app.jsonResponse(w, 200, ret)
}

func (app *APP) getPopularPostHandler(w http.ResponseWriter, r *http.Request) {
	cachePosts, err := app.feedCache.PopularFeed.Get(app.ctx)
	// previously we cached popular feed periodically in background
	if err != nil {
		log.Fatal("Error in get popular feed from cache: ", err)
		return
	} else if cachePosts != nil {
		postIDs := make([]uint32, len(cachePosts))
		for i, post := range cachePosts {
			postIDs[i] = post.ID
		}
		ret := GetPostPayload{PostIds: postIDs, UserId: 0}
		app.jsonResponse(w, 200, ret)
	} else {
		log.Fatal("Error: cache does not exist")
	}
}

func (app *APP) getRandomPostHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := app.feed.Posts.GetRandomFeed(app.ctx, 20)
	if err != nil {
		log.Fatal(err)
		return
	}

	postIDs := make([]uint32, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}
	ret := GetPostPayload{UserId: 0, PostIds: postIDs}
	app.jsonResponse(w, 200, ret)
}
