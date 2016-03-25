package main

import (
	"net/http"
)

func saveFirstRunHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	registerUser(session, r.FormValue("email"), r.FormValue("password"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func firstRunHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()
	c := getUserCollection(session)
	num, err := c.Count()
	if num != 0 && err == nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, r, "firstRun", nil)
}
