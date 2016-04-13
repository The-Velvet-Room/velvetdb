package main

import (
	"sort"
	r "github.com/dancannon/gorethink"
)

type TournamentResult struct {
	ID         string `gorethink:"id,omitempty"`
	Tournament string `gorethink:"tournament"`
	Player     string `gorethink:"player"`
	Seed       int    `gorethink:"seed"`
	Place      int    `gorethink:"placement"`
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
	}).EqJoin("tournament", getTournamentTable()).Zip().
	OrderBy("date_start").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}

	var results []*TournamentResult
	err = c.All(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}
