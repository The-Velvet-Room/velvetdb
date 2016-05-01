package main

import (
	"fmt"
	"net/http"

	r "gopkg.in/dancannon/gorethink.v2"
)

type GameType struct {
	ID      string `gorethink:"id,omitempty"`
	Name    string `gorethink:"name"`
	URLPath string `gorethink:"urlpath"`
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

func fetchGameType(ID string) (*GameType, error) {
	c, err := getGameTypeTable().Get(ID).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return nil, err
	}
	var gt *GameType
	err = c.One(&gt)
	if err != nil {
		return nil, err
	}
	return gt, nil
}

func fetchGameTypeByURLPath(p string) (*GameType, error) {
	c, err := getGameTypeTable().Filter(map[string]interface{}{
		"urlpath": p,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	var gt *GameType
	err = c.One(&gt)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return gt, nil
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
	urlpath := r.FormValue("urlpath")
	addGameType(GameType{Name: name, URLPath: urlpath})
	http.Redirect(w, r, "/", http.StatusFound)
}
