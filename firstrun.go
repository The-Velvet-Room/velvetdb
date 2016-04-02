package main

import (
	"net/http"
)

func usersExist() bool {
	c, err := getUserTable().Count().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		return false
	}
	var count int
	err = c.One(&count)
	if err != nil {
		return false
	}
	return count != 0
}

func saveFirstRunHandler(w http.ResponseWriter, r *http.Request) {
	if usersExist() {
		http.NotFound(w, r)
		return
	}
	registerUser(r.FormValue("email"), r.FormValue("password"), getMaxPermissionLevel())
	http.Redirect(w, r, "/", http.StatusFound)
}

func firstRunHandler(w http.ResponseWriter, r *http.Request) {
	if usersExist() {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, r, "firstRun", nil)
}
