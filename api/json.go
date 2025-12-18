package main

import (
	"encoding/json"
	"net/http"
)

type httpErrorResponse struct {
	ErrorMessage string `json:"error"`
}

const MAX_SIZE_READ int64 = 1_048_578

func readJSON(writer http.ResponseWriter, reader *http.Request, data any) error {
	http.MaxBytesReader(writer, reader.Body, MAX_SIZE_READ)
	decoder := json.NewDecoder(reader.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(data)
}

func jsonResponse(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) error {
	return jsonResponse(w, status, httpErrorResponse{ErrorMessage: message})
}
