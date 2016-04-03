package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	r "github.com/dancannon/gorethink"

	"time"

	"github.com/aspic/go-challonge"
	"github.com/gorilla/mux"
)

type Tournament struct {
	ID          string    `gorethink:"id,omitempty"`
	GameType    string    `gorethink:"gametype"`
	Name        string    `gorethink:"name"`
	BracketURL  string    `gorethink:"bracket_url"`
	VODUrl      string    `gorethink:"vod_url"`
	DateStart   time.Time `gorethink:"date_start"`
	DateEnd     time.Time `gorethink:"date_end"`
	PlayerCount	int		  `gorethink:"player_count"`
	Editing bool    `gorethink:"editing"`
}

const initialLastID string = "0"

type matchesByID []*challonge.Match

func (a matchesByID) Len() int           { return len(a) }
func (a matchesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a matchesByID) Less(i, j int) bool { return a[i].Id < a[j].Id }

func getTournamentTable() r.Term {
	return r.Table("tournaments")
}

func updateTournamentEditing(id string, editing bool) {
	_, err := getTournamentTable().Get(id).Update(map[string]interface{}{
		"editing": editing,
	}).RunWrite(dataStore.GetSession())

	if err != nil {
		fmt.Println(err)
	}
}

func addTournament(t Tournament) string {
	wr, err := getTournamentTable().Insert(&t).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	return wr.GeneratedKeys[0]
}

func fetchTournament(id string) (*Tournament, error) {
	c, err := getTournamentTable().Get(id).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var t Tournament
	err = c.One(&t)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &t, nil
}

func fetchTournaments(gametype string, inProgress bool) (*[]Tournament, error) {
	query := getTournamentTable()
	if gametype != "" {
		query = query.Filter(map[string]interface{}{
			"gametype": gametype,
		})
	}
	if !inProgress {
		query = query.Filter(map[string]interface{}{
			"last_match_id": "",
		})
	}
	c, err := query.OrderBy("date_start").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var t []Tournament
	err = c.All(&t)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &t, nil
}

func addTournamentHandler(w http.ResponseWriter, r *http.Request) {
	gameTypes := fetchGameTypes()

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

	c, err := getTournamentTable().Filter(map[string]interface{}{
		"bracket_url": url,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	// if we find a tournament with the same bracket url,
	// redirect to that tournament
	if !c.IsNil() {
		var t Tournament
		c.One(&t)
		http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
		return
	}

	t := fetchChallongeTournament(url)

	id := addTournament(Tournament{
		Name:       name,
		BracketURL: url,
		GameType:   gametype,
		DateStart:  *t.StartedAt,
		DateEnd:    *t.UpdatedAt,
		PlayerCount: t.ParticipantsCount,
		Editing: true,
	})

	http.Redirect(w, r, "/tournament/"+id, http.StatusFound)
}

func addTournamentMatchesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ct := fetchChallongeTournament(t.BracketURL)
	players := fetchPlayers()
	data := struct {
		Tournament *Tournament
		Participants []*challonge.Participant
		Players []Player
	} {
		t,
		ct.Participants,
		players,
	}
	renderTemplate(w, r, "addTournamentMatch", data)
}

func saveTournamentMatches2Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	playerMap := make(map[string]string)
	r.ParseForm()
	for k, v := range r.PostForm {
		split := strings.Split(k, "_")
		if split[0] != "p" {
			continue
		}
		var playerID string
		if v[0] == "new" {
			playerID = addPlayer(Player{
				Nickname: r.FormValue("newname_" + k),
			})
		} else {
			playerID = r.FormValue("select_" + k)
		}
		playerMap[split[1]] = playerID
	}

	for k, v := range playerMap {
		fmt.Println(k, v)
	}

	matches := fetchChallongeMatches(t.BracketURL)
	newGames := []*Game{}
	for _, m := range matches {
		scoreSplit := strings.Split(m.Scores, "-")
		m.PlayerOneScore, _ = strconv.Atoi(scoreSplit[0])
		m.PlayerTwoScore, _ = strconv.Atoi(scoreSplit[1])
		p1 := strconv.Itoa(m.PlayerOneId)
		p2 := strconv.Itoa(m.PlayerTwoId)
		tMatchID := strconv.Itoa(m.Id)
		newGames = append(newGames, &Game{
			Date: *m.UpdatedAt,
			GameType: t.GameType,
			Tournament: t.ID,
			TournamentMatchID: tMatchID,
			Player1: playerMap[p1],
			Player2: playerMap[p2],
			Player1score: m.PlayerOneScore,
			Player2score: m.PlayerTwoScore,
		})
	}
	_, err = getGameTable().Insert(newGames).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	updateTournamentEditing(t.ID, false)
	http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
}

func viewTournamentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if t.Editing {
		if _, ok := isLoggedIn(r); ok {
			http.Redirect(w, r, "/tournament/addmatches/"+t.ID, http.StatusFound)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	games := fetchGamesForTournament(t.ID)
	players := fetchPlayers()
	playerMap := make(map[string]Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	data := struct {
		Tournament *Tournament
		Games      []Game
		PlayerMap  map[string]Player
	}{
		t,
		games,
		playerMap,
	}

	renderTemplate(w, r, "viewTournament", data)
}

func viewTournamentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameType := vars["gametype"]

	var gameTypes []GameType
	var selectedType *GameType
	var t *[]Tournament
	if gameType == "" {
		gameTypes = fetchGameTypes()
	} else {
		var err error
		selectedType, err = fetchGameTypeByURLPath(gameType)
		if err != nil {
			http.Redirect(w, r, "/tournaments", http.StatusTemporaryRedirect)
			return
		}
		_, showInProgress := isLoggedIn(r)
		t, _ = fetchTournaments(selectedType.ID, showInProgress)
	}

	data := struct {
		Tournaments      *[]Tournament
		GameTypes        []GameType
		SelectedGameType *GameType
	}{
		t,
		gameTypes,
		selectedType,
	}

	renderTemplate(w, r, "tournaments", data)
}

func saveTournamentMatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	matchID := r.FormValue("match")
	matches := fetchChallongeMatches(t.BracketURL)

	intID, _ := strconv.Atoi(matchID)
	var match *challonge.Match
	for index := range matches {
		if matches[index].Id == intID {
			match = matches[index]
			break
		}
	}

	player1 := r.FormValue("player1")
	var player1ID string
	if player1 == "new" {
		player1ID = addPlayer(Player{
			Nickname: r.FormValue("player1newname"),
		})
	} else {
		player1ID = r.FormValue("player1select")
	}

	player2 := r.FormValue("player2")
	var player2ID string
	if player2 == "new" {
		player2ID = addPlayer(Player{
			Nickname: r.FormValue("player2newname"),
		})
	} else {
		player2ID = r.FormValue("player2select")
	}

	player1score, _ := strconv.Atoi(r.FormValue("player1score"))
	player2score, _ := strconv.Atoi(r.FormValue("player2score"))

	addGame(Game{
		Tournament:        t.ID,
		TournamentMatchID: matchID,
		Date:              *match.UpdatedAt,
		GameType:          t.GameType,
		Player1:           player1ID,
		Player2:           player2ID,
		Player1score:      player1score,
		Player2score:      player2score,
	})

	http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
}

func fetchChallongeTournament(url string) *challonge.Tournament {
	client := challonge.New(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey)
	hash := parseChallongeURL(url)
	tourneyData, err := client.NewTournamentRequest(hash).WithParticipants().Get()
	if err != nil {
		fmt.Println(err)
	}
	return tourneyData
}

func fetchChallongeMatches(url string) []*challonge.Match {
	client := challonge.New(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey)
	hash := parseChallongeURL(url)
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
