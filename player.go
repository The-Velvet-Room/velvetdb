package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	r "github.com/dancannon/gorethink"
	"github.com/dancannon/gorethink/types"
	"github.com/gorilla/mux"
	"github.com/jasonwinn/geocoder"
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
	if (city != player.City || state != player.State) && city != "" && state != "" {
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

	matches := fetchMatchesForPlayer(player.ID, canEdit)
	results, _ := fetchResultsForPlayer(player.ID)

	data := struct {
		Player    *Player
		Matches   []Match
		Results   []*TournamentResult
		PlayerMap map[string]Player
		CanEdit   bool
	}{
		player,
		matches,
		results,
		playerMap,
		canEdit,
	}

	renderTemplate(w, r, "player", data)
}
