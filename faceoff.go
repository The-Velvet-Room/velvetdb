package main

import (
	"net/http"
)

func faceoffHandler(w http.ResponseWriter, r *http.Request) {
	p := fetchPlayers()
	p1 := r.FormValue("p1")
	p2 := r.FormValue("p2")
	var games *[]Game
	var player1 Player
	var player2 Player
	if p1 != "" && p2 != "" {
		for _, pl := range p {
			if p1 == pl.URLPath {
				player1 = pl
			}
			if p2 == pl.URLPath {
				player2 = pl
			}
		}
		games = fetchGamesForPlayers(player1.ID, player2.ID)
	}
	data := struct {
		Players []Player
		Games   *[]Game
		Player1 *Player
		Player2 *Player
	}{
		p,
		games,
		&player1,
		&player2,
	}
	renderTemplate(w, r, "faceoff", data)
}
