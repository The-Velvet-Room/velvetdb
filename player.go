package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
)

type Player struct {
	ID         string   `gorethink:"id,omitempty"`
	Nickname   string   `gorethink:"nickname"`
	Tag        string   `gorethink:"tag"`
	Aliases    []string `gorethink:"aliases"`
	Image      string   `gorethink:"image"`
	URLPath    string   `gorethink:"urlpath"`
	FirstName  string   `gorethink:"first_name"`
	LastName   string   `gorethink:"last_name"`
	Facts      []string `gorethink:"facts"`
	Characters []string `gorethink:"characters"`
	Twitter    string   `gorethink:"twitter"`
	Twitch     string   `gorethink:"twitch"`
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

	urlpath := r.FormValue("urlpath")
	if urlpath != player.URLPath {
		player, _ := fetchPlayerByURLPath(urlpath)
		if player != nil {
			http.Redirect(w, r, "/editplayer/"+playerNick, http.StatusBadRequest)
			return
		}
	}

	_, err = getPlayerTable().Filter(map[string]interface{}{
		"urlpath": playerNick,
	}).Update(map[string]interface{}{
		"nickname":   r.FormValue("nickname"),
		"urlpath":    urlpath,
		"tag":        r.FormValue("tag"),
		"first_name": r.FormValue("firstname"),
		"last_name":  r.FormValue("lastname"),
	}).RunWrite(dataStore.GetSession())
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

	data := struct {
		Player    *Player
		Matches   []Match
		PlayerMap map[string]Player
		CanEdit   bool
	}{
		player,
		matches,
		playerMap,
		canEdit,
	}

	renderTemplate(w, r, "player", data)
}
