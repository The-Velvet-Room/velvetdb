package main

import (
	"net/http"
)

func saveFirstRunHandler(w http.ResponseWriter, r *http.Request) {
	registerUser(r.FormValue("email"), r.FormValue("password"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func firstRunHandler(w http.ResponseWriter, r *http.Request) {
	var count int
	c, err := getUserTable().Count().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = c.One(&count)
	if err != nil || count != 0 {
		http.NotFound(w, r)
		return
	}

	renderTemplate(w, r, "firstRun", nil)
}
