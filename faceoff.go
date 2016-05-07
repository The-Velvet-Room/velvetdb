package main

import (
	"net/http"
)

func faceoffHandler(w http.ResponseWriter, r *http.Request) {
	p1 := r.FormValue("p1")
	p2 := r.FormValue("p2")

	type GameTypeMatches struct {
		GameType     *GameType
		Player1Sets  int
		Player2Sets  int
		Player1Games int
		Player2Games int
		Matches      []Match
	}

	gameMatches := []GameTypeMatches{}
	tournamentMap := map[string]*Tournament{}
	var player1 *Player
	var player2 *Player
	if p1 != "" && p2 != "" {
		var err1 error
		var err2 error
		player1, err1 = fetchPlayerByURLPath(p1)
		player2, err2 = fetchPlayerByURLPath(p2)
		if err1 != nil || err2 != nil {
			http.Redirect(w, r, "/faceoff", http.StatusFound)
		}

		_, logged := isLoggedIn(r)
		matches := fetchMatchesForPlayers(player1.ID, player2.ID, logged)

		// sort matches by game
		gameIndex := map[string]int{}
		for _, m := range *matches {
			if _, ok := gameIndex[m.GameType]; !ok {
				gt, _ := fetchGameType(m.GameType)
				gameMatches = append(gameMatches, GameTypeMatches{
					GameType: gt,
					Matches:  []Match{},
				})
				gameIndex[m.GameType] = len(gameMatches) - 1
			}
			gameMatches[gameIndex[m.GameType]].Matches = append(gameMatches[gameIndex[m.GameType]].Matches, m)
		}

		for idx := range gameMatches {
			for _, g := range gameMatches[idx].Matches {
				if g.Player1 == player1.ID {
					gameMatches[idx].Player1Games += g.Player1score
					gameMatches[idx].Player2Games += g.Player2score
					if g.Player1score > g.Player2score {
						gameMatches[idx].Player1Sets++
					} else {
						gameMatches[idx].Player2Sets++
					}
				} else {
					gameMatches[idx].Player2Games += g.Player1score
					gameMatches[idx].Player1Games += g.Player2score
					if g.Player1score > g.Player2score {
						gameMatches[idx].Player2Sets++
					} else {
						gameMatches[idx].Player1Sets++
					}
				}
			}
		}
		// build tournament map
		ts, _ := fetchTournamentsForPlayers(p1, p2)
		for _, t := range ts {
			tournamentMap[t.ID] = t
		}
	}
	data := struct {
		Matches       []GameTypeMatches
		TournamentMap map[string]*Tournament
		Player1       *Player
		Player2       *Player
	}{
		gameMatches,
		tournamentMap,
		player1,
		player2,
	}
	renderTemplate(w, r, "faceoff", data)
}
