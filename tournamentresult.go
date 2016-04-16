package main

import (
	"sort"

	r "github.com/dancannon/gorethink"
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
		OrderBy(r.Desc("right.date_start")).Run(dataStore.GetSession())
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
