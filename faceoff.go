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
	player1sets, player2sets, player1games, player2games := 0, 0, 0, 0
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
		for _, g := range *games {
			if g.Player1 == player1.ID {
				player1games += g.Player1score
				player2games += g.Player2score
				if g.Player1score > g.Player2score {
					player1sets++
				} else {
					player2sets++
				}
			} else {
				player2games += g.Player1score
				player1games += g.Player2score
				if g.Player1score > g.Player2score {
					player2sets++
				} else {
					player1sets++
				}
			}
		}
	}
	data := struct {
		Players      []Player
		Games        *[]Game
		Player1      *Player
		Player2      *Player
		Player1Sets  int
		Player2Sets  int
		Player1Games int
		Player2Games int
	}{
		p,
		games,
		&player1,
		&player2,
		player1sets,
		player2sets,
		player1games,
		player2games,
	}
	renderTemplate(w, r, "faceoff", data)
}
