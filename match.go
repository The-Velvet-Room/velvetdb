package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	r "github.com/dancannon/gorethink"
)

type Match struct {
	ID                string    `gorethink:"id,omitempty"`
	Tournament        string    `gorethink:"tournament,omitempty"`
	TournamentMatchID string    `gorethink:"tournament_match_id"`
	GameType          string    `gorethink:"gametype"`
	Date              time.Time `gorethink:"date"`
	Player1           string    `gorethink:"player1"`
	Player2           string    `gorethink:"player2"`
	Player1score      int       `gorethink:"player1_score"`
	Player2score      int       `gorethink:"player2_score"`
	Round             int       `gorethink:"round"`
}

func getMatchTable() r.Term {
	return r.Table("matches")
}

func fetchMatchesForPlayer(id string) []Match {
	c, err := getMatchTable().Filter(r.Or(
		r.Row.Field("player1").Eq(id),
		r.Row.Field("player2").Eq(id),
	)).OrderBy(r.Desc("date")).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	matches := []Match{}
	err = c.All(&matches)
	if err != nil {
		fmt.Println(err)
	}
	return matches
}

func fetchMatchesForPlayers(p1 string, p2 string) *[]Match {
	c, err := getMatchTable().Filter(r.Or(
		r.Row.Field("player1").Eq(p1).And(r.Row.Field("player2").Eq(p2)),
		r.Row.Field("player1").Eq(p2).And(r.Row.Field("player2").Eq(p1)),
	)).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	matches := []Match{}
	err = c.All(&matches)
	if err != nil {
		fmt.Println(err)
	}
	return &matches
}

func fetchMatchesForTournament(id string) []Match {
	c, err := getMatchTable().Filter(map[string]interface{}{
		"tournament": id,
	}).OrderBy("date").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	matches := []Match{}
	err = c.All(&matches)
	if err != nil {
		fmt.Println(err)
	}
	return matches
}

func addMatch(m Match) string {
	wr, err := getMatchTable().Insert(m).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	return wr.GeneratedKeys[0]
}

func addMatchHandler(w http.ResponseWriter, r *http.Request) {
	// get players
	players := fetchPlayers()

	gameTypes := fetchGameTypes()

	data := struct {
		Players   []Player
		GameTypes []GameType
	}{
		players,
		gameTypes,
	}

	renderTemplate(w, r, "addMatch", data)
}

func saveMatchHandler(w http.ResponseWriter, r *http.Request) {
	player1 := r.FormValue("player1")
	player2 := r.FormValue("player2")
	player1score, _ := strconv.Atoi(r.FormValue("player1score"))
	player2score, _ := strconv.Atoi(r.FormValue("player2score"))
	gameType := r.FormValue("gametype")
	if player1 == player2 {
		http.Redirect(w, r, "/", http.StatusBadRequest)
	}

	addMatch(Match{
		Player1:      player1,
		Player2:      player2,
		GameType:     gameType,
		Player1score: player1score,
		Player2score: player2score,
		Date:         time.Now(),
	})

	http.Redirect(w, r, "/", http.StatusFound)
}