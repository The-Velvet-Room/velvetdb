package main

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aspic/go-challonge"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Tournament struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	GameType bson.ObjectId
	Name     string
	URL      string
	LastID   string
}

const initialLastID string = "0"

type matchesByID []*challonge.Match

func (a matchesByID) Len() int           { return len(a) }
func (a matchesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a matchesByID) Less(i, j int) bool { return a[i].Id < a[j].Id }

func getTournamentCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("tournaments")
}

func updateTournamentLastID(session *mgo.Session, id bson.ObjectId, lastID string) {
	c := getTournamentCollection(session)
	updateErr := c.Update(bson.M{"_id": id}, bson.M{"$set": bson.M{"lastid": lastID}})
	if updateErr != nil {
		fmt.Println(updateErr)
	}
}

func addTournamentHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	gameTypes := fetchGameTypes(session)

	data := struct {
		GameTypes []GameType
	}{
		gameTypes,
	}

	renderTemplate(w, r, "addTournament", data)
}

func saveTournamentHandler(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	name := r.FormValue("name")
	gametype := r.FormValue("gametype")

	session := dataStore.GetSession()
	defer session.Close()

	c := getTournamentCollection(session)
	var tournament Tournament
	oneErr := c.Find(bson.M{"url": url}).One(&tournament)
	if oneErr != nil {
		fmt.Println(oneErr)
	}

	err := c.Insert(&Tournament{
		Name:     name,
		URL:      url,
		LastID:   initialLastID,
		GameType: bson.ObjectIdHex(gametype),
	})
	if err != nil {
		fmt.Println(err)
	}
	oneErr = c.Find(bson.M{"url": url}).One(&tournament)
	if oneErr != nil {
		fmt.Println(oneErr)
	}

	http.Redirect(w, r, "/tournament/"+tournament.ID.Hex(), http.StatusFound)
}

func viewTournamentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]
	if !bson.IsObjectIdHex(tournamentID) {
		http.NotFound(w, r)
		return
	}
	session := dataStore.GetSession()
	defer session.Close()

	c := getTournamentCollection(session)
	var tournament Tournament
	findErr := c.Find(bson.M{"_id": bson.ObjectIdHex(tournamentID)}).One(&tournament)
	if findErr != nil {
		http.NotFound(w, r)
		return
	}

	if tournament.LastID == "" {
		games := fetchGamesForTournament(session, tournament.ID)
		players := fetchPlayers(session)
		playerMap := make(map[bson.ObjectId]Player)
		for _, p := range players {
			playerMap[p.ID] = p
		}

		data := struct {
			Tournament Tournament
			Games      []Game
			PlayerMap  map[bson.ObjectId]Player
		}{
			tournament,
			games,
			playerMap,
		}

		renderTemplate(w, r, "viewTournament", data)
		return
	}

	matches := fetchChallongeMatches(tournament.URL)

	intID, _ := strconv.Atoi(tournament.LastID)
	var match *challonge.Match
	var count int
	nextIter := false

	if tournament.LastID == initialLastID {
		count = 1
		match = matches[0]
	} else {
		for index, m := range matches {
			if nextIter {
				match = m
				count = index + 1
				break
			}
			if m.Id == intID {
				nextIter = true
			}
		}
	}

	// we've loaded all matches
	if match == nil {
		updateTournamentLastID(session, tournament.ID, "")
		games := fetchGamesForTournament(session, tournament.ID)
		data := struct {
			Tournament Tournament
			Games      []Game
		}{
			tournament,
			games,
		}

		renderTemplate(w, r, "viewTournament", data)
		return
	}

	scoreSplit := strings.Split(match.Scores, "-")
	match.PlayerOneScore, _ = strconv.Atoi(scoreSplit[0])
	match.PlayerTwoScore, _ = strconv.Atoi(scoreSplit[1])

	players := fetchPlayers(session)

	data := struct {
		MatchNumber int
		MatchCount  int
		Match       *challonge.Match
		Tournament  Tournament
		Players     []Player
	}{
		count,
		len(matches),
		match,
		tournament,
		players,
	}

	renderTemplate(w, r, "addTournamentMatch", data)
}

func saveTournamentMatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	if !bson.IsObjectIdHex(tournamentID) {
		http.NotFound(w, r)
		return
	}
	session := dataStore.GetSession()
	defer session.Close()

	c := getTournamentCollection(session)
	var tournament Tournament
	findErr := c.Find(bson.M{"_id": bson.ObjectIdHex(tournamentID)}).One(&tournament)
	if findErr != nil {
		http.NotFound(w, r)
		return
	}

	matchID := r.FormValue("match")
	matches := fetchChallongeMatches(tournament.URL)

	intID, _ := strconv.Atoi(matchID)
	var match *challonge.Match
	for index := range matches {
		if matches[index].Id == intID {
			match = matches[index]
			break
		}
	}

	player1 := r.FormValue("player1")
	var player1ID bson.ObjectId
	if player1 == "new" {
		player1ID = addPlayer(session, Player{
			Nickname: match.PlayerOne.Name,
		})
	} else {
		player1ID = bson.ObjectIdHex(r.FormValue("player1select"))
	}

	player2 := r.FormValue("player2")
	var player2ID bson.ObjectId
	if player2 == "new" {
		player2ID = addPlayer(session, Player{
			Nickname: match.PlayerTwo.Name,
		})
	} else {
		player2ID = bson.ObjectIdHex(r.FormValue("player2select"))
	}

	player1score, _ := strconv.Atoi(r.FormValue("player1score"))
	player2score, _ := strconv.Atoi(r.FormValue("player2score"))

	addGame(session, Game{
		Tournament:        tournament.ID,
		TournamentMatchID: matchID,
		Date:              *match.UpdatedAt,
		GameType:          tournament.GameType,
		Player1:           player1ID,
		Player2:           player2ID,
		Player1score:      player1score,
		Player2score:      player2score,
	})

	updateTournamentLastID(session, bson.ObjectIdHex(tournamentID), matchID)

	http.Redirect(w, r, "/tournament/"+tournament.ID.Hex(), http.StatusFound)
}

func fetchChallongeMatches(url string) []*challonge.Match {
	client := challonge.New(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey)
	hash := parseChallongeURL(url)
	fmt.Println(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey, hash)
	tourneyData, err := client.NewTournamentRequest(hash).WithMatches().WithParticipants().Get()
	if err != nil {
		fmt.Println(err)
	}
	matches := tourneyData.GetMatches()
	sort.Sort(matchesByID(matches))
	return matches
}

func parseChallongeURL(url string) string {
	tourneyHash := url[strings.LastIndex(url, "/")+1 : len(url)]
	tourneyHash = strings.TrimSpace(tourneyHash)

	//If tournament belongs to an organization,
	//it must be specified in the request
	if len(strings.Split(url, "."))-1 > 1 {
		orgHash := url[strings.LastIndex(url, "http://")+7 : strings.Index(url, ".")]
		challongeHash := orgHash + "-" + tourneyHash
		return challongeHash
	}

	//Standard tournament
	return tourneyHash
}
