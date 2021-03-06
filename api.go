package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
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

func handleAPIGameTypes(w http.ResponseWriter, r *http.Request) {
	gt := fetchGameTypes()
	writeAPIResponse(w, r, gt)
}

func handleAPIPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerID := vars["id"]

	p, err := fetchPlayer(playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAPIResponse(w, r, p)
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

func handleAPIPlayerTournamentResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerID := vars["id"]

	gameType := r.FormValue("gametype")

	rs, err := fetchResultsForPlayer(playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	type ResultJSON struct {
		Seed       int         `json:"seed"`
		Place      int         `json:"place"`
		Tournament *Tournament `json:"tournament"`
	}

	results := []ResultJSON{}
	for _, result := range rs {
		if gameType != "" && gameType != result.Tournament.GameType {
			continue
		}
		results = append(results, ResultJSON{
			Seed:       result.Seed,
			Place:      result.Place,
			Tournament: result.Tournament,
		})
	}

	writeAPIResponse(w, r, results)
}

func handleAPIPlayerMatches(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerID := vars["id"]

	gametype := r.FormValue("gametype")

	matches := fetchMatchesForPlayer(playerID, false)

	filterMatches := []Match{}
	for _, m := range matches {
		if gametype != "" && gametype != m.GameType {
			continue
		}
		filterMatches = append(filterMatches, m)
	}
	writeAPIResponse(w, r, filterMatches)
}

func handleAPIFaceoff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	p1 := vars["p1"]
	p2 := vars["p2"]

	gametype := r.FormValue("gametype")

	matches := fetchMatchesForPlayers(p1, p2, false)

	filterMatches := []Match{}
	for _, m := range *matches {
		if gametype != "" && gametype != m.GameType {
			continue
		}
		filterMatches = append(filterMatches, m)
	}
	writeAPIResponse(w, r, filterMatches)
}
