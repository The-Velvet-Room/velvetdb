package main

import (
	r "github.com/dancannon/gorethink"
)

type TournamentResult struct {
	ID         string `gorethink:"id,omitempty"`
	Tournament string `gorethink:"tournament"`
	Player     string `gorethink:"player"`
	Seed       int    `gorethink:"seed"`
	Place      int    `gorethink:"placement"`
}

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
	return results, nil
}
