package main

import (
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type GameType struct {
	ID   bson.ObjectId `bson:"_id,omitempty"`
	Name string
}

func fetchGameTypes(session *mgo.Session) []GameType {
	c := session.DB("test").C("gametypes")
	gameTypeIter := c.Find(nil).Iter()

	gameTypes := []GameType{}
	var gameTypeResult *GameType
	for gameTypeIter.Next(&gameTypeResult) {
		gameTypes = append(gameTypes, *gameTypeResult)
	}
	return gameTypes
}

func addGameTypeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "addGameType", nil)
}

func saveGameTypeHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()
	name := r.FormValue("name")
	c := session.DB("test").C("gametypes")
	saveErr := c.Insert(&GameType{Name: name})
	if saveErr != nil {
		fmt.Println(saveErr)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
