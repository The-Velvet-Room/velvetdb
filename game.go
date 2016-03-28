package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	r "github.com/dancannon/gorethink"
)

type Game struct {
	ID                string `gorethink:"id,omitempty"`
	Tournament        string `gorethink:"tournament,omitempty"`
	TournamentMatchID string `gorethink:"tournament_match_id"`
	GameType          string `gorethink:"gametype"`
	Date              time.Time `gorethink:"date"`
	Player1           string `gorethink:"player1"`
	Player2           string `gorethink:"player2"`
	Player1score      int `gorethink:"player1_score"`
	Player2score      int`gorethink:"player2_score"`
}

func getGameTable() r.Term {
	return r.Table("games")
}

func fetchGamesForPlayer(id string) []Game {
	c, err := getGameTable().Filter(r.Or(
		r.Row.Field("player1").Eq(id),
		r.Row.Field("player2").Eq(id),
	)).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	games := []Game{}
	err = c.All(&games)
	if err != nil {
		fmt.Println(err)
	}
	return games
}

func fetchGamesForTournament(id string) []Game {
	c, err := getGameTable().Filter(map[string]interface{}{
		"tournament": id,
	}).OrderBy("date").Run(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	games := []Game{}
	err = c.All(&games)
	if err != nil {
		fmt.Println(err)
	}
	return games
}

func addGame(game Game) {
	wr, err := getGameTable().Insert(game).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
}

func saveGameHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	player1 := r.FormValue("player1")
	player2 := r.FormValue("player2")
	player1score, _ := strconv.Atoi(r.FormValue("player1score"))
	player2score, _ := strconv.Atoi(r.FormValue("player2score"))
	gameType := r.FormValue("gametype")
	if player1 == player2 {
		http.Redirect(w, r, "/", http.StatusBadRequest)
	}

	addGame(Game{
		Player1:      player1,
		Player2:      player2,
		GameType:     gameType,
		Player1score: player1score,
		Player2score: player2score,
		Date:         time.Now(),
	})

	http.Redirect(w, r, "/", http.StatusFound)
}
