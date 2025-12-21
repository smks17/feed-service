package main

import (
	"context"
	"errors"
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
	route.Use(middleware.Timeout(10 * time.Second))
	route.Use(middleware.Recoverer)

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

func (app *APP) UpdatePopularFeedCache(ctx context.Context) {
	if err := app.updatePopularFeed(ctx); err != nil {
		log.Println("Error updating popular feed:", err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := app.updatePopularFeed(ctx); err != nil {
				log.Println("Error updating popular feed:", err)
			}
		case <-app.ctx.Done():
			fmt.Println("Stop signal received, exiting goroutine.")
			return
		}
	}
}

func (app *APP) updatePopularFeed(parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	dbPosts, err := app.feed.Posts.GetPopularFeed(ctx)
	if err != nil {
		return err
	}
	err = app.feedCache.PopularFeed.Set(ctx, dbPosts)
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
		app.internalServerError(w, r, err)
		return
	}

	var posts []feed.Post

	ctx := r.Context()
	cachePosts, err := app.feedCache.HomeFeed.Get(ctx, uint32(userID))
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("error in get feeds of user %d from cache: %v", userID, err))
		return
	} else if cachePosts != nil {
		posts = cachePosts
		log.Println("Hit cache for for user ", userID)
	} else {
		dbPosts, err := app.feed.Posts.GetHomeFeed(ctx, uint32(userID))
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
		// set cache in background
		go func() {
			ctx, cancel := context.WithTimeout(app.ctx, 5*time.Second)
			defer cancel()

			err := app.feedCache.HomeFeed.Set(ctx, uint32(userID), dbPosts)
			if err != nil {
				log.Println("error in set feeds of user ", userID, " from cache: ", err)
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
	jsonResponse(w, 200, ret)
}

func (app *APP) getExplorePostHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	var posts []feed.Post
	ctx := r.Context()

	cachePosts, err := app.feedCache.ExploreFeed.Get(ctx, uint32(userID))
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("error in get feeds of user %d from cache: %v", userID, err))
		return
	} else if cachePosts != nil {
		posts = cachePosts
		log.Println("Hit cache for for user ", userID)
	} else {
		dbPosts, err := app.feed.Posts.GetExploreFeed(ctx, uint32(userID), app.feedCache.PopularFeed.Get)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
		// set cache in background
		go func() {
			err = app.feedCache.ExploreFeed.Set(app.ctx, uint32(userID), dbPosts)
			if err != nil {
				log.Println("error in set feeds of user", userID, "from cache: ", err)
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
	jsonResponse(w, 200, ret)
}

func (app *APP) getPopularPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cachePosts, err := app.feedCache.PopularFeed.Get(ctx)
	// previously we cached popular feed periodically in background
	if err != nil {
		app.internalServerError(w, r, err)
		return
	} else if cachePosts != nil {
		postIDs := make([]uint32, len(cachePosts))
		for i, post := range cachePosts {
			postIDs[i] = post.ID
		}
		ret := GetPostPayload{PostIds: postIDs, UserId: 0}
		jsonResponse(w, 200, ret)
	} else {
		app.internalServerError(w, r, errors.New("error: cache does not exist"))
	}
}

func (app *APP) getRandomPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	posts, err := app.feed.Posts.GetRandomFeed(ctx, 20)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	postIDs := make([]uint32, len(posts))
	for i, post := range posts {
		postIDs[i] = post.ID
	}
	ret := GetPostPayload{UserId: 0, PostIds: postIDs}
	jsonResponse(w, 200, ret)
}
