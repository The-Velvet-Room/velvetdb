package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jasonwinn/geocoder"
	r "gopkg.in/dancannon/gorethink.v2"
	"gopkg.in/dancannon/gorethink.v2/types"
)

type Player struct {
	ID         string      `gorethink:"id,omitempty"`
	Nickname   string      `gorethink:"nickname"`
	Tag        string      `gorethink:"tag"`
	Aliases    []string    `gorethink:"aliases"`
	Image      string      `gorethink:"image"`
	URLPath    string      `gorethink:"urlpath"`
	FirstName  string      `gorethink:"first_name"`
	LastName   string      `gorethink:"last_name"`
	Facts      []string    `gorethink:"facts"`
	Characters []string    `gorethink:"characters"`
	Twitter    string      `gorethink:"twitter"`
	Twitch     string      `gorethink:"twitch"`
	City       string      `gorethink:"city"`
	State      string      `gorethink:"state"`
	Location   types.Point `gorethink:"location,omitempty"`
}

var alphanumeric = regexp.MustCompile("[^A-Za-z0-9]+")

func getPlayerTable() r.Term {
	return r.Table("players")
}

func addPlayer(player Player) string {
	if player.URLPath == "" {
		player.URLPath = strings.ToLower(alphanumeric.ReplaceAllString(player.Nickname, ""))
		if player.URLPath == "" {
			player.ID = dataStore.GetID()
			player.URLPath = player.ID
		} else {
			_, err := fetchPlayerByURLPath(player.URLPath)
			// if we find a player, set the URLPath to the id
			if err == nil {
				player.ID = dataStore.GetID()
				player.URLPath = player.ID
			}
		}
	}
	wr, err := getPlayerTable().Insert(player).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	if len(wr.GeneratedKeys) != 0 {
		return wr.GeneratedKeys[0]
	}
	return player.ID
}

func fetchPlayer(id string) (*Player, error) {
	c, err := getPlayerTable().Get(id).Run(dataStore.GetSession())
	defer c.Close()

	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var player *Player
	err = c.One(&player)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return player, nil
}

func fetchPlayerByNickname(nickname string) (*Player, error) {
	c, err := getPlayerTable().Filter(map[string]interface{}{
		"nickname": nickname,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var player *Player
	err = c.One(&player)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return player, nil
}

func fetchPlayerByURLPath(urlpath string) (*Player, error) {
	c, err := getPlayerTable().Filter(map[string]interface{}{
		"urlpath": urlpath,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var player *Player
	err = c.One(&player)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return player, nil
}

func fetchPlayersSearch(nickname string) ([]*Player, error) {
	c, err := getPlayerTable().
		Filter(r.Row.Field("nickname").Match("(?i)" + nickname)).
		OrderBy("nickname").Limit(10).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}
	players := []*Player{}
	err = c.All(&players)
	if err != nil {
		return nil, err
	}
	return players, nil
}

func fetchPlayers() []Player {
	c, err := getPlayerTable().OrderBy("nickname").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	players := []Player{}
	err = c.All(&players)
	if err != nil {
		fmt.Println(err)
	}
	return players
}

func addPlayerHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "addPlayer", nil)
}

func savePlayerHandler(w http.ResponseWriter, r *http.Request) {
	n := r.FormValue("nickname")
	addPlayer(Player{Nickname: n})
	http.Redirect(w, r, "/", http.StatusFound)
}

func saveEditPlayerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerNick := vars["playerNick"]

	player, err := fetchPlayerByURLPath(playerNick)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	r.ParseForm()
	urlpath := r.FormValue("urlpath")
	if urlpath != player.URLPath {
		player, _ := fetchPlayerByURLPath(urlpath)
		if player != nil {
			http.Redirect(w, r, "/editplayer/"+playerNick, http.StatusBadRequest)
			return
		}
	}

	city := r.FormValue("city")
	state := r.FormValue("state")

	point := player.Location
	// if city or state are different and they're not empty, run the geocoder
	if city == "" && state == "" {
		point = types.Point{}
	} else if city != player.City || state != player.State {
		geocoder.SetAPIKey(siteConfiguration.MapquestApiKey)
		lat, lng, err := geocoder.Geocode(city + "," + state)
		if err == nil {
			point.Lat = lat
			point.Lon = lng
		}
	}

	facts := []string{}
	for _, v := range r.Form["facts"] {
		if v != "" {
			facts = append(facts, v)
		}
	}
	characters := []string{}
	for _, v := range r.Form["characters"] {
		if v != "" {
			characters = append(characters, v)
		}
	}
	aliases := []string{}
	for _, v := range r.Form["aliases"] {
		if v != "" {
			aliases = append(aliases, v)
		}
	}

	update := map[string]interface{}{
		"nickname":   r.FormValue("nickname"),
		"urlpath":    urlpath,
		"tag":        r.FormValue("tag"),
		"image":      r.FormValue("image"),
		"first_name": r.FormValue("firstname"),
		"last_name":  r.FormValue("lastname"),
		"city":       city,
		"state":      state,
		"twitter":    r.FormValue("twitter"),
		"twitch":     r.FormValue("twitch"),
		"facts":      facts,
		"aliases":    aliases,
		"characters": characters,
		"location":   point,
	}

	_, err = getPlayerTable().Filter(map[string]interface{}{
		"urlpath": playerNick,
	}).Update(update).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	http.Redirect(w, r, "/player/"+urlpath, http.StatusFound)
}

func editPlayerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerNick := vars["playerNick"]

	player, err := fetchPlayerByURLPath(playerNick)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Player *Player
	}{
		player,
	}

	renderTemplate(w, r, "editPlayer", data)
}

func playersHandler(w http.ResponseWriter, r *http.Request) {
	players := fetchPlayers()
	_, loggedIn := isLoggedIn(r)

	data := struct {
		Players  []Player
		LoggedIn bool
	}{
		players,
		loggedIn,
	}
	renderTemplate(w, r, "players", data)
}

func playerViewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerNick := vars["playerNick"]

	player, err := fetchPlayerByURLPath(playerNick)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	players := fetchPlayers()
	playerMap := make(map[string]Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	_, canEdit := isLoggedIn(r)

	type GameTypeMatches struct {
		GameType *GameType
		Matches  []Match
	}

	matches := fetchMatchesForPlayer(player.ID, canEdit)

	gameIndex := map[string]int{}
	gameMatches := []GameTypeMatches{}
	for _, m := range matches {
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

	type GameTypeResults struct {
		GameType *GameType
		Results  []*TournamentResult
	}

	results, _ := fetchResultsForPlayer(player.ID)

	resultIndex := map[string]int{}
	tournamentMap := map[string]*Tournament{}
	gameResults := []GameTypeResults{}
	for _, result := range results {
		tournamentMap[result.TournamentID] = result.Tournament
		if _, ok := resultIndex[result.Tournament.GameType]; !ok {
			gt, _ := fetchGameType(result.Tournament.GameType)
			gameResults = append(gameResults, GameTypeResults{
				GameType: gt,
				Results:  []*TournamentResult{},
			})
			resultIndex[result.Tournament.GameType] = len(gameResults) - 1
		}
		gameResults[resultIndex[result.Tournament.GameType]].Results = append(gameResults[resultIndex[result.Tournament.GameType]].Results, result)
	}

	data := struct {
		Player        *Player
		Matches       []GameTypeMatches
		Results       []GameTypeResults
		PlayerMap     map[string]Player
		TournamentMap map[string]*Tournament
		CanEdit       bool
	}{
		player,
		gameMatches,
		gameResults,
		playerMap,
		tournamentMap,
		canEdit,
	}

	renderTemplate(w, r, "player", data)
}

func mergePlayersHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "mergePlayers", nil)
}

func saveMergePlayersHandler(w http.ResponseWriter, r *http.Request) {
	keepPlayerID := r.FormValue("playerkeep")
	mergePlayerID := r.FormValue("playermerge")
	if keepPlayerID == mergePlayerID || keepPlayerID == "" || mergePlayerID == "" {
		http.Redirect(w, r, "/players", http.StatusFound)
		return
	}

	// merge matches into the player to keep
	getMatchTable().Filter(map[string]interface{}{
		"player1": mergePlayerID,
	}).Update(map[string]interface{}{
		"player1": keepPlayerID,
	}).RunWrite(dataStore.GetSession())

	getMatchTable().Filter(map[string]interface{}{
		"player2": mergePlayerID,
	}).Update(map[string]interface{}{
		"player2": keepPlayerID,
	}).RunWrite(dataStore.GetSession())

	// merge tournament results into the player to keep
	getTournamentResultTable().Filter(map[string]interface{}{
		"player": mergePlayerID,
	}).Update(map[string]interface{}{
		"player": keepPlayerID,
	}).RunWrite(dataStore.GetSession())

	// delete the player
	getPlayerTable().Get(mergePlayerID).Delete().RunWrite(dataStore.GetSession())

	http.Redirect(w, r, "/players", http.StatusFound)
}
