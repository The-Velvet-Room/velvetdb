package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	r "gopkg.in/dancannon/gorethink.v2"

	"github.com/dguenther/go-challonge"
	"github.com/gorilla/mux"
)

type Tournament struct {
	ID          string    `gorethink:"id,omitempty"`
	GameType    string    `gorethink:"gametype"`
	Name        string    `gorethink:"name"`
	BracketURL  string    `gorethink:"bracket_url"`
	VODUrl      string    `gorethink:"vod_url"`
	PoolOf      string    `gorethink:"pool_of"`
	DateStart   time.Time `gorethink:"date_start"`
	DateEnd     time.Time `gorethink:"date_end"`
	PlayerCount int       `gorethink:"player_count"`
	Editing     bool      `gorethink:"editing"`
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

func fetchTournaments(gametype string, editing bool) (*[]Tournament, error) {
	query := getTournamentTable()
	filter := map[string]interface{}{
		"pool_of": "",
	}

	if gametype != "" {
		filter["gametype"] = gametype
	}
	if !editing {
		filter["editing"] = false
	}
	c, err := query.Filter(filter).OrderBy(r.Desc("date_start")).Run(dataStore.GetSession())
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

func fetchTournamentPools(ID string) ([]*Tournament, error) {
	query := getTournamentTable()
	c, err := query.Filter(map[string]interface{}{
		"pool_of": ID,
	}).OrderBy("name").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var t []*Tournament
	err = c.All(&t)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return t, nil
}

func addPoolHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]
	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Tournament *Tournament
	}{
		t,
	}

	renderTemplate(w, r, "addPool", data)
}

func savePoolHandler(w http.ResponseWriter, r *http.Request) {
	poolOf := r.FormValue("poolOf")
	name := r.FormValue("name")
	url := r.FormValue("url")

	c, err := getTournamentTable().Filter(map[string]interface{}{
		"bracket_url": url,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	// if we find a tournament with the same bracket url,
	// redirect to that tournament
	var t Tournament
	if !c.IsNil() {
		c.One(&t)
		http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
		return
	}

	ct := fetchChallongeTournament(url)

	id := addTournament(Tournament{
		Name:        name,
		BracketURL:  url,
		PoolOf:      poolOf,
		GameType:    t.GameType,
		DateStart:   *ct.StartedAt,
		DateEnd:     *ct.UpdatedAt,
		PlayerCount: ct.ParticipantsCount,
		Editing:     true,
	})

	http.Redirect(w, r, "/tournament/"+id, http.StatusFound)
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
		Name:        name,
		BracketURL:  url,
		GameType:    gametype,
		DateStart:   *t.StartedAt,
		DateEnd:     *t.UpdatedAt,
		PlayerCount: t.ParticipantsCount,
		Editing:     true,
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
		Tournament   *Tournament
		Participants []*challonge.Participant
		Complete     bool
		Players      []Player
	}{
		t,
		ct.Participants,
		ct.State == "complete",
		players,
	}
	renderTemplate(w, r, "addTournamentMatch", data)
}

func saveTournamentMatchesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Find the ID of the root tournament
	var rootTournamentID string
	if t.PoolOf == "" {
		rootTournamentID = t.ID
	} else {
		ct, _ := fetchTournament(t.PoolOf)
		for ct.PoolOf != "" {
			ct, _ = fetchTournament(ct.PoolOf)
		}
		rootTournamentID = ct.ID
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

	ct := fetchChallongeTournament(t.BracketURL)

	// Add tournament results
	oldResults, _ := fetchResultsForTournament(rootTournamentID)
	resultDict := map[string]*TournamentResult{}
	for _, r := range oldResults {
		resultDict[r.Player] = r
	}

	newResults := []*TournamentResult{}
	for _, p := range ct.Participants {
		id := strconv.Itoa(p.Id)
		// If we've already saved a result for this player,
		// don't save another one
		if _, ok := resultDict[playerMap[id]]; ok {
			continue
		}

		// Add tournament results for all players we haven't added yet.
		tr := &TournamentResult{
			TournamentID: rootTournamentID,
			Player:       playerMap[id],
			Place:        0,
			Seed:         0,
		}
		// Only add the place and seed of the player if it's the final bracket.
		if t.PoolOf == "" {
			tr.Place = p.FinalRank
			tr.Seed = p.Seed
		}
		newResults = append(newResults, tr)
	}
	_, err = getTournamentResultTable().Insert(newResults).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}

	// Add tournament matches
	newMatches := []*Match{}
	for _, m := range ct.Matches {
		// Only add completed matches
		if m.State != "complete" {
			continue
		}
		scoreSplit := strings.SplitN(m.Scores, "-", 2)
		m.PlayerOneScore, _ = strconv.Atoi(scoreSplit[0])
		m.PlayerTwoScore, _ = strconv.Atoi(scoreSplit[1])
		p1 := strconv.Itoa(m.PlayerOneId)
		p2 := strconv.Itoa(m.PlayerTwoId)
		var p1prereq *string
		var p2prereq *string
		if m.PlayerOnePrereqMatch != nil {
			p1prereq = new(string)
			*p1prereq = strconv.Itoa(*m.PlayerOnePrereqMatch)
		}
		if m.PlayerTwoPrereqMatch != nil {
			p2prereq = new(string)
			*p2prereq = strconv.Itoa(*m.PlayerTwoPrereqMatch)
		}
		tMatchID := strconv.Itoa(m.Id)
		newMatches = append(newMatches, &Match{
			Date:                       *m.UpdatedAt,
			GameType:                   t.GameType,
			Tournament:                 t.ID,
			TournamentMatchID:          tMatchID,
			Player1:                    playerMap[p1],
			Player2:                    playerMap[p2],
			Player1PrevTournamentMatch: p1prereq,
			Player2PrevTournamentMatch: p2prereq,
			Player1score:               m.PlayerOneScore,
			Player2score:               m.PlayerTwoScore,
			Round:                      m.Round,
		})
	}
	_, err = getMatchTable().Insert(newMatches).RunWrite(dataStore.GetSession())
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

	_, logged := isLoggedIn(r)

	matches := fetchMatchesForTournament(t.ID, logged)
	players := fetchPlayers()
	playerMap := make(map[string]Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	var poolOf *Tournament
	if t.PoolOf != "" {
		poolOf, _ = fetchTournament(t.PoolOf)
	}

	pools, _ := fetchTournamentPools(t.ID)
	results, _ := fetchResultsForTournament(t.ID)

	data := struct {
		Tournament *Tournament
		PoolOf     *Tournament
		Pools      []*Tournament
		Results    []*TournamentResult
		Matches    []Match
		PlayerMap  map[string]Player
		IsLoggedIn bool
	}{
		t,
		poolOf,
		pools,
		results,
		matches,
		playerMap,
		logged,
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

func fetchChallongeTournament(url string) *challonge.Tournament {
	client := challonge.New(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey)
	hash := parseChallongeURL(url)
	tourneyData, err := client.NewTournamentRequest(hash).WithMatches().WithParticipants().Get()
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
