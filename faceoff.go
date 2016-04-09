package main

import (
	"net/http"
)

func faceoffHandler(w http.ResponseWriter, r *http.Request) {
	p := fetchPlayers()
	p1 := r.FormValue("p1")
	p2 := r.FormValue("p2")
	var matches *[]Match
	var player1 Player
	var player2 Player
	player1sets, player2sets, player1matches, player2matches := 0, 0, 0, 0
	if p1 != "" && p2 != "" {
		for _, pl := range p {
			if p1 == pl.URLPath {
				player1 = pl
			}
			if p2 == pl.URLPath {
				player2 = pl
			}
		}
		matches = fetchMatchesForPlayers(player1.ID, player2.ID)
		for _, g := range *matches {
			if g.Player1 == player1.ID {
				player1matches += g.Player1score
				player2matches += g.Player2score
				if g.Player1score > g.Player2score {
					player1sets++
				} else {
					player2sets++
				}
			} else {
				player2matches += g.Player1score
				player1matches += g.Player2score
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
		Matches      *[]Match
		Player1      *Player
		Player2      *Player
		Player1Sets  int
		Player2Sets  int
		Player1Games int
		Player2Games int
	}{
		p,
		matches,
		&player1,
		&player2,
		player1sets,
		player2sets,
		player1matches,
		player2matches,
	}
	renderTemplate(w, r, "faceoff", data)
}
