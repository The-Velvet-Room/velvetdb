package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Player struct {
	ID         bson.ObjectId `bson:"_id,omitempty"`
	Nickname   string
	Tag        string
	Aliases    []string
	Image      string
	URLPath    string
	FirstName  string
	LastName   string
	Facts      []string
	Characters []string
}

var alphanumeric = regexp.MustCompile("[^A-Za-z0-9]+")

func getPlayerCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("players")
}

func addPlayer(session *mgo.Session, player Player) bson.ObjectId {
	c := getPlayerCollection(session)
	if !player.ID.Valid() {
		player.ID = bson.NewObjectId()
	}
	if player.URLPath == "" {
		player.URLPath = strings.ToLower(alphanumeric.ReplaceAllString(player.Nickname, ""))
		if player.URLPath == "" {
			player.URLPath = player.ID.Hex()
		} else {
			_, err := fetchPlayerByURLPath(session, player.URLPath)
			// if we find a player, set the URLPath to the id
			if err == nil {
				player.URLPath = player.ID.Hex()
			}
		}
	}
	saveErr := c.Insert(player)
	if saveErr != nil {
		fmt.Println(saveErr)
	}
	return player.ID
}

func fetchPlayer(session *mgo.Session, id bson.ObjectId) (*Player, error) {
	c := getPlayerCollection(session)
	var player *Player
	err := c.FindId(id).One(&player)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return player, nil
}

func fetchPlayerByNickname(session *mgo.Session, nickname string) (*Player, error) {
	c := getPlayerCollection(session)
	var player *Player
	err := c.Find(bson.M{"nickname": nickname}).One(&player)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return player, nil
}

func fetchPlayerByURLPath(session *mgo.Session, urlpath string) (*Player, error) {
	c := getPlayerCollection(session)
	var player *Player
	err := c.Find(bson.M{"urlpath": urlpath}).One(&player)
	if err != nil {
		return nil, err
	}
	return player, nil
}

func fetchPlayers(session *mgo.Session) []Player {
	c := getPlayerCollection(session)
	playerIter := c.Find(nil).Sort("nickname").Iter()

	players := []Player{}
	var result *Player
	for playerIter.Next(&result) {
		players = append(players, *result)
	}
	return players
}

func playerViewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	playerNick := vars["playerNick"]

	session := dataStore.GetSession()
	defer session.Close()

	player, err := fetchPlayerByURLPath(session, playerNick)
	if err == mgo.ErrNotFound {
		http.NotFound(w, r)
		return
	}

	players := fetchPlayers(session)
	playerMap := make(map[bson.ObjectId]Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	games := fetchGamesForPlayer(session, player.ID)

	data := struct {
		Player    *Player
		Games     []Game
		PlayerMap map[bson.ObjectId]Player
	}{
		player,
		games,
		playerMap,
	}

	renderTemplate(w, r, "player", data)
}
