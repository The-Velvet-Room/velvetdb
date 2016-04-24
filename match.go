package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	r "gopkg.in/dancannon/gorethink.v2"
)

type Match struct {
	ID                         string    `gorethink:"id,omitempty"`
	Tournament                 string    `gorethink:"tournament,omitempty"`
	TournamentMatchID          string    `gorethink:"tournament_match_id"`
	Player1PrevTournamentMatch *string   `gorethink:"p1_prev_tm"`
	Player2PrevTournamentMatch *string   `gorethink:"p2_prev_tm"`
	GameType                   string    `gorethink:"gametype"`
	Date                       time.Time `gorethink:"date"`
	Player1                    string    `gorethink:"player1"`
	Player2                    string    `gorethink:"player2"`
	Player1score               int       `gorethink:"player1_score"`
	Player2score               int       `gorethink:"player2_score"`
	Round                      int       `gorethink:"round"`
	Hidden                     bool      `gorethink:"hidden"`
}

func getMatchTable() r.Term {
	return r.Table("matches")
}

func fetchMatch(id string) (*Match, error) {
	c, err := getMatchTable().Get(id).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}
	var m *Match
	err = c.One(&m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func fetchMatchesForPlayer(id string, includeHidden bool) []Match {
	filter := r.Or(r.Row.Field("player1").Eq(id), r.Row.Field("player2").Eq(id))
	if !includeHidden {
		filter = r.And(r.Row.Field("hidden").Eq(false), filter)
	}

	c, err := getMatchTable().Filter(filter).
		OrderBy(r.Desc("date")).Run(dataStore.GetSession())
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

func fetchMatchesForPlayers(p1 string, p2 string, includeHidden bool) *[]Match {
	filter := r.Or(
		r.Row.Field("player1").Eq(p1).And(r.Row.Field("player2").Eq(p2)),
		r.Row.Field("player1").Eq(p2).And(r.Row.Field("player2").Eq(p1)),
	)
	if !includeHidden {
		r.And(filter, r.Row.Field("hidden").Eq(false))
	}
	c, err := getMatchTable().Filter(filter).Run(dataStore.GetSession())
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

func fetchMatchesForTournament(id string, includeHidden bool) []Match {
	filter := map[string]interface{}{"tournament": id}
	if !includeHidden {
		filter["hidden"] = false
	}

	c, err := getMatchTable().Filter(filter).OrderBy("date").Run(dataStore.GetSession())
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

func editMatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["match"]

	players := fetchPlayers()

	m, err := fetchMatch(matchID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Match   *Match
		Players []Player
		Saved   bool
	}{
		m,
		players,
		false,
	}
	renderTemplate(w, r, "editMatch", data)
}

func saveEditMatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["match"]

	_, err := fetchMatch(matchID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	p1 := r.FormValue("p1")
	p2 := r.FormValue("p2")
	p1score, _ := strconv.Atoi(r.FormValue("p1score"))
	p2score, _ := strconv.Atoi(r.FormValue("p2score"))
	hidden := r.FormValue("hidden") == "hidden"

	wr, werr := getMatchTable().Get(matchID).Update(map[string]interface{}{
		"hidden":        hidden,
		"player1":       p1,
		"player2":       p2,
		"player1_score": p1score,
		"player2_score": p2score,
	}).RunWrite(dataStore.GetSession())
	if werr != nil {
		fmt.Println(err)
	}
	if wr.Errors > 0 {
		fmt.Println(wr.FirstError)
	}

	newMatch, err := fetchMatch(matchID)
	if err != nil {
		fmt.Println(err)
	}

	data := struct {
		Match   *Match
		Players []Player
		Saved   bool
	}{
		newMatch,
		fetchPlayers(),
		true,
	}
	renderTemplate(w, r, "editMatch", data)
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
