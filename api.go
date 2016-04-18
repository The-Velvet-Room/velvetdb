package main

import (
	"encoding/json"
	"net/http"
)

func writeAPIResponse(w http.ResponseWriter, data interface{}) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	if err := json.NewEncoder(buf).Encode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	buf.WriteTo(w)
}

func handleAPIPlayers(w http.ResponseWriter, r *http.Request) {
	p := fetchPlayers()
	writeAPIResponse(w, p)
}
