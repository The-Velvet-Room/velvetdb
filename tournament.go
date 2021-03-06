package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dguenther/go-bracket"
	"github.com/gorilla/mux"
	"github.com/jasonwinn/geocoder"
	r "gopkg.in/dancannon/gorethink.v2"
	"gopkg.in/dancannon/gorethink.v2/types"
)

type Tournament struct {
	ID          string      `gorethink:"id,omitempty"`
	GameType    string      `gorethink:"gametype"`
	Name        string      `gorethink:"name"`
	BracketURL  string      `gorethink:"bracket_url"`
	VODUrl      string      `gorethink:"vod_url"`
	PoolOf      string      `gorethink:"pool_of"`
	DateStart   time.Time   `gorethink:"date_start"`
	DateEnd     time.Time   `gorethink:"date_end"`
	City        string      `gorethink:"city"`
	State       string      `gorethink:"state"`
	Location    types.Point `gorethink:"location,omitempty"`
	PlayerCount int         `gorethink:"player_count"`
	Editing     bool        `gorethink:"editing"`
}

const initialLastID string = "0"

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

func fetchTournamentsForPlayer(ID string) ([]*Tournament, error) {
	c, err := getMatchTable().Filter(
		r.Or(r.Row.Field("player1").Eq(ID), r.Row.Field("player2").Eq(ID)),
	).EqJoin("tournament", getTournamentTable()).
		Without("left").Field("right").Distinct().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}

	var t []*Tournament
	err = c.All(&t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func fetchTournamentsForPlayers(player1 string, player2 string) ([]*Tournament, error) {
	c, err := getMatchTable().Filter(
		r.Or(
			r.And(r.Row.Field("player1").Eq(player1), r.Row.Field("player2").Eq(player2)),
			r.And(r.Row.Field("player1").Eq(player2), r.Row.Field("player2").Eq(player1)),
		),
	).EqJoin("tournament", getTournamentTable()).
		Without("left").Field("right").Distinct().Run(dataStore.GetSession())
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

func deleteTournament(ID string) {
	// Delete the tournament matches
	getMatchTable().Filter(map[string]interface{}{
		"tournament": ID,
	}).Delete().RunWrite(dataStore.GetSession())
	// Delete the tournament results
	getTournamentResultTable().Filter(map[string]interface{}{
		"tournament": ID,
	}).Delete().RunWrite(dataStore.GetSession())
	// Delete the tournament
	getTournamentTable().Get(ID).Delete().RunWrite(dataStore.GetSession())
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
	var t *Tournament
	if !c.IsNil() {
		c.One(&t)
		http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
		return
	}

	t, err = fetchTournament(poolOf)
	if err != nil {
		http.Error(w, "No tournament with ID "+poolOf+" found", http.StatusBadRequest)
		return
	}

	ct := fetchExternalBracket(url)

	id := addTournament(Tournament{
		Name:        name,
		BracketURL:  url,
		PoolOf:      poolOf,
		GameType:    t.GameType,
		City:        t.City,
		State:       t.State,
		Location:    t.Location,
		DateStart:   *ct.StartedAt,
		DateEnd:     *ct.UpdatedAt,
		PlayerCount: len(ct.Players),
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

func editTournamentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	gameTypes := fetchGameTypes()

	data := struct {
		Tournament *Tournament
		GameTypes  []GameType
	}{
		t,
		gameTypes,
	}

	renderTemplate(w, r, "editTournament", data)
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

	city := r.FormValue("city")
	state := r.FormValue("state")
	point := types.Point{}
	if city != "" && state != "" {
		geocoder.SetAPIKey(siteConfiguration.MapquestApiKey)
		lat, lng, err := geocoder.Geocode(city + "," + state)
		if err == nil {
			point.Lat = lat
			point.Lon = lng
		}
	}

	t := fetchExternalBracket(url)

	id := addTournament(Tournament{
		Name:        name,
		BracketURL:  url,
		GameType:    gametype,
		DateStart:   *t.StartedAt,
		DateEnd:     *t.UpdatedAt,
		PlayerCount: len(t.Players),
		City:        city,
		State:       state,
		Location:    point,
		Editing:     true,
	})

	http.Redirect(w, r, "/tournament/"+id, http.StatusFound)
}

func saveEditTournamentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	city := r.FormValue("city")
	state := r.FormValue("state")

	point := t.Location
	// if city or state are different and they're not empty, run the geocoder
	if city == "" && state == "" {
		point = types.Point{}
	} else if city != t.City || state != t.State {
		geocoder.SetAPIKey(siteConfiguration.MapquestApiKey)
		lat, lng, err := geocoder.Geocode(city + "," + state)
		if err == nil {
			point.Lat = lat
			point.Lon = lng
		}
	}

	wr, err := getTournamentTable().Get(tournamentID).Update(map[string]interface{}{
		"name":     r.FormValue("name"),
		"gametype": r.FormValue("gametype"),
		"city":     city,
		"state":    state,
		"location": point,
	}).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	if wr.Errors > 0 {
		fmt.Println(wr.FirstError)
	}
	http.Redirect(w, r, "/tournament/"+t.ID, http.StatusFound)
}

func addTournamentMatchesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	t, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ct := fetchExternalBracket(t.BracketURL)
	data := struct {
		Tournament   *Tournament
		Participants []*bracket.Player
		Complete     bool
	}{
		t,
		ct.Players,
		ct.State == "complete",
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

	b := fetchExternalBracket(t.BracketURL)

	// Add tournament results
	oldResults, _ := fetchResultsForTournament(rootTournamentID)
	resultDict := map[string]*TournamentResult{}
	for _, r := range oldResults {
		resultDict[r.Player] = r
	}

	newResults := []*TournamentResult{}
	for _, p := range b.Players {
		// If we've already saved a result for this player,
		// don't save another one
		ourPlayerID := playerMap[p.ID]
		if _, ok := resultDict[ourPlayerID]; ok {
			continue
		}

		// Add tournament results for all players we haven't added yet.
		tr := &TournamentResult{
			TournamentID: rootTournamentID,
			Player:       ourPlayerID,
			Place:        0,
			Seed:         0,
		}
		// Only add the place and seed of the player if it's the final bracket.
		if t.PoolOf == "" {
			tr.Place = p.Rank
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
	for _, m := range b.Matches {
		// Only add completed matches
		if m.State != "complete" {
			continue
		}

		newMatches = append(newMatches, &Match{
			Date:                       *m.UpdatedAt,
			GameType:                   t.GameType,
			Tournament:                 t.ID,
			TournamentMatchID:          m.ID,
			Player1:                    playerMap[m.Player1ID],
			Player2:                    playerMap[m.Player2ID],
			Player1PrevTournamentMatch: m.Player1PrereqMatchID,
			Player2PrevTournamentMatch: m.Player2PrereqMatchID,
			Player1score:               m.Player1Score,
			Player2score:               m.Player2Score,
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

func deleteTournamentHandler(w http.ResponseWriter, r *http.Request) {
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
	renderTemplate(w, r, "deleteTournament", data)
}

func saveDeleteTournamentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tournamentID := vars["tournament"]

	_, err := fetchTournament(tournamentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	deleteTournament(tournamentID)
	http.Redirect(w, r, "/", http.StatusFound)
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

	gametype, _ := fetchGameType(t.GameType)
	pools, _ := fetchTournamentPools(t.ID)
	results, _ := fetchResultsForTournament(t.ID)
	// Sort the results into results that have a place and results that don't
	placedResults := []*TournamentResult{}
	unplacedResults := []*TournamentResult{}
	for _, r := range results {
		if r.Place == 0 {
			unplacedResults = append(unplacedResults, r)
		} else {
			placedResults = append(placedResults, r)
		}
	}

	data := struct {
		Tournament      *Tournament
		GameType        *GameType
		PoolOf          *Tournament
		Pools           []*Tournament
		PlacedResults   []*TournamentResult
		UnplacedResults []*TournamentResult
		Matches         []Match
		PlayerMap       map[string]Player
		IsLoggedIn      bool
	}{
		t,
		gametype,
		poolOf,
		pools,
		placedResults,
		unplacedResults,
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

func fetchExternalBracket(url string) *bracket.Bracket {
	client := bracket.NewClient(siteConfiguration.ChallongeDevUsername, siteConfiguration.ChallongeApiKey)
	bracket, err := client.FetchBracket(url)
	if err != nil {
		fmt.Println(err)
	}
	return bracket
}
