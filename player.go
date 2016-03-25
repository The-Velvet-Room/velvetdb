package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Player struct {
	ID         bson.ObjectId `bson:"_id,omitempty"`
	Tag        string
	Aliases    []string
	Nickname   string
	URLStub    string
	FirstName  string
	LastName   string
	Facts      []string
	Characters []string
}

func getPlayerCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("players")
}

func addPlayer(session *mgo.Session, player Player) bson.ObjectId {
	c := getPlayerCollection(session)
	if !player.ID.Valid() {
		player.ID = bson.NewObjectId()
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
	playerID := vars["player"]

	if !bson.IsObjectIdHex(playerID) {
		http.NotFound(w, r)
		return
	}
	session := dataStore.GetSession()
	defer session.Close()

	bsonPlayerID := bson.ObjectIdHex(playerID)

	player, err := fetchPlayer(session, bsonPlayerID)
	if err == mgo.ErrNotFound {
		http.NotFound(w, r)
		return
	}

	players := fetchPlayers(session)
	playerMap := make(map[bson.ObjectId]Player)
	for _, p := range players {
		playerMap[p.ID] = p
	}

	games := fetchGamesForPlayer(session, bsonPlayerID)

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
