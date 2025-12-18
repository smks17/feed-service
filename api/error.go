package main

import (
	"log"
	"net/http"
)

func (app *APP) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Println("internal error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusInternalServerError, "the server has a problem")
}
