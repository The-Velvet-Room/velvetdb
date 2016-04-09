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
