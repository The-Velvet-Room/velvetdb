package main

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
)

func rankPlayers(gameType string) map[string]*EloDict {
	players := fetchPlayers()
	playerDict := make(map[string]Player)
	rankDict := make(map[string]*EloDict)

	for _, p := range players {
		playerDict[p.ID] = p
	}

	c, err := getMatchTable().Filter(map[string]interface{}{
		"gametype": gameType,
		"hidden":   false,
	}).OrderBy("date").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	e := &Elo{k: 32}

	var m Match
	for c.Next(&m) {
		if _, ok := rankDict[m.Player1]; !ok {
			rankDict[m.Player1] = NewEloDict(playerDict[m.Player1])
		}
		if _, ok := rankDict[m.Player2]; !ok {
			rankDict[m.Player2] = NewEloDict(playerDict[m.Player2])
		}
		expectedScore1 := e.getExpected(rankDict[m.Player1].Rank, rankDict[m.Player2].Rank)
		expectedScore2 := e.getExpected(rankDict[m.Player2].Rank, rankDict[m.Player1].Rank)

		player1results := float64(m.Player1score) / float64(m.Player1score+m.Player2score)

		rankDict[m.Player1].Rank = e.updateRating(expectedScore1, player1results, rankDict[m.Player1].Rank)
		rankDict[m.Player2].Rank = e.updateRating(expectedScore2, 1-player1results, rankDict[m.Player2].Rank)
	}
	return rankDict
}

func rankingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameType := vars["gametype"]

	var gameTypes []GameType
	var selectedType *GameType
	var ranks []*EloDict
	if gameType == "" {
		gameTypes = fetchGameTypes()
	} else {
		var err error
		selectedType, err = fetchGameTypeByURLPath(gameType)
		if err != nil {
			http.Redirect(w, r, "/rankings", http.StatusTemporaryRedirect)
			return
		}
		rankDict := rankPlayers(selectedType.ID)

		ranks = make([]*EloDict, len(rankDict))
		idx := 0
		for _, v := range rankDict {
			ranks[idx] = v
			idx++
		}

		sort.Sort(ByRank(ranks))

	}
	data := struct {
		Ranks            []*EloDict
		GameTypes        []GameType
		SelectedGameType *GameType
	}{
		ranks,
		gameTypes,
		selectedType,
	}
	renderTemplate(w, r, "rankings", data)
}
