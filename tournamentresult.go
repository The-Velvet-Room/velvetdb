package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	r "gopkg.in/dancannon/gorethink.v2"
)

type TournamentResult struct {
	ID           string      `gorethink:"id,omitempty"`
	TournamentID string      `gorethink:"tournament"`
	Tournament   *Tournament `gorethink:"-"`
	Player       string      `gorethink:"player"`
	Seed         int         `gorethink:"seed"`
	Place        int         `gorethink:"placement"`
}

type ByPlace []*TournamentResult

func (a ByPlace) Len() int           { return len(a) }
func (a ByPlace) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPlace) Less(i, j int) bool { return a[i].Place < a[j].Place }

func getTournamentResultTable() r.Term {
	return r.Table("tournamentresults")
}

func fetchTournamentResult(resultID string) (*TournamentResult, error) {
	c, err := getTournamentResultTable().Get(resultID).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}

	var result *TournamentResult
	err = c.One(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func fetchResultsForTournament(tournamentID string) ([]*TournamentResult, error) {
	c, err := getTournamentResultTable().Filter(map[string]interface{}{
		"tournament": tournamentID,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}

	var results []*TournamentResult
	err = c.All(&results)
	if err != nil {
		return nil, err
	}

	sort.Sort(ByPlace(results))
	return results, nil
}

func fetchResultsForPlayer(playerID string) ([]*TournamentResult, error) {
	c, err := getTournamentResultTable().Filter(map[string]interface{}{
		"player": playerID,
	}).EqJoin("tournament", getTournamentTable()).
		OrderBy(r.Desc(r.Row.Field("right").Field("date_start"))).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}

	type joinType struct {
		Left  *TournamentResult
		Right *Tournament
	}
	var result joinType
	results := []*TournamentResult{}
	for c.Next(&result) {
		result.Left.Tournament = result.Right
		results = append(results, result.Left)
	}
	if err != nil {
		return nil, err
	}

	return results, nil
}

func fetchTournamentsForPlayers(player1 string, player2 string) ([]*Tournament, error) {
	c, err := getTournamentResultTable().Filter(
		r.Or(r.Row.Field("player").Eq(player1), r.Row.Field("player").Eq(player2)),
	).EqJoin("tournament", getTournamentTable()).
		OrderBy(r.Desc(r.Row.Field("right").Field("date_start"))).Without("left").
		Field("right").Distinct().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}
	matches := []*Tournament{}
	err = c.All(&matches)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func editTournamentResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resultID := vars["result"]
	result, err := fetchTournamentResult(resultID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tournament, _ := fetchTournament(result.TournamentID)
	player, _ := fetchPlayer(result.Player)

	data := struct {
		TournamentResult *TournamentResult
		Tournament       *Tournament
		Player           *Player
	}{
		result,
		tournament,
		player,
	}
	renderTemplate(w, r, "editTournamentResult", data)
}

func saveEditTournamentResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resultID := vars["result"]
	result, err := fetchTournamentResult(resultID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	seed, _ := strconv.Atoi(r.FormValue("seed"))
	place, _ := strconv.Atoi(r.FormValue("place"))

	wr, err := getTournamentResultTable().Get(resultID).Update(map[string]interface{}{
		"seed":      seed,
		"placement": place,
	}).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	if wr.Errors > 0 {
		fmt.Println(wr.FirstError)
	}
	http.Redirect(w, r, "/tournament/"+result.TournamentID, http.StatusFound)
}
