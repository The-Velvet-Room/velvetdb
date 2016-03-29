package main

import (
	"fmt"
	"net/http"

	r "github.com/dancannon/gorethink"
)

type GameType struct {
	ID   string `gorethink:"id,omitempty"`
	Name string `gorethink:"name"`
}

func getGameTypeTable() r.Term {
	return r.Table("gametypes")
}

func fetchGameTypes() []GameType {
	c, err := getGameTypeTable().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	gameTypes := []GameType{}
	err = c.All(&gameTypes)
	if err != nil {
		fmt.Println(err)
	}
	return gameTypes
}

func addGameType(gameType GameType) string {
	wr, err := getGameTypeTable().Insert(gameType).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	return wr.GeneratedKeys[0]
}

func addGameTypeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "addGameType", nil)
}

func saveGameTypeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	addGameType(GameType{Name: name})
	http.Redirect(w, r, "/", http.StatusFound)
}
