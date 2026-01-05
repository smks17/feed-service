package main

import (
	"log"
	"net/http"

	"github.com/smks17/feed-service/lib/env"
)

func InternalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Internal-Token")
		value, err := env.CheckEnv("FEED_SERVICE_TOKEN")
		if err != nil {
			log.Fatal("Feed Service Token is not defined!")
		}
		if token != value {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
