package main

import (
	"encoding/json"
	"net/http"
)

func writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func writeAPIResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	if err := json.NewEncoder(buf).Encode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writeCORSHeaders(w, r)
	buf.WriteTo(w)
}

func handleAPIPreflight(w http.ResponseWriter, r *http.Request) {
	writeCORSHeaders(w, r)
}

func handleAPIPlayers(w http.ResponseWriter, r *http.Request) {
	p := fetchPlayers()
	writeAPIResponse(w, r, p)
}

func handleAPIPlayersSearch(w http.ResponseWriter, r *http.Request) {
	search := r.FormValue("query")
	p, err := fetchPlayersSearch(search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAPIResponse(w, r, p)
}
