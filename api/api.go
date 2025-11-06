package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

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
	config APPConfig
	feed   *feed.Feed
	// httServer *http.Server
}

func newApp(feed *feed.Feed, config APPConfig) *APP {

	return &APP{config: config, feed: feed}
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
	User    uint32 `json:"user"`
	Content string `json:"content" validate:"required,max=1000"`
}

func (app *APP) getHomePostHandler(w http.ResponseWriter, r *http.Request) {
	idParam, err := strconv.Atoi(chi.URLParam(r, "userID"))
	if err != nil {
		log.Fatal(err)
		return
	}

	posts, err := app.feed.Posts.GetHomeFeed(uint32(idParam))
	if err != nil {
		log.Fatal(err)
		return
	}

	ret := make([]GetPostPayload, len(posts))
	for i, post := range posts {
		ret[i] = GetPostPayload{User: post.Author, Content: post.Content}
	}
	app.jsonResponse(w, 200, ret)
}
