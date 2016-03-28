package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/boj/rethinkstore"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              string `gorethink:"id,omitempty"`
	Email           string `gorethink:"email"`
	Player          string `gorethink:"player,omitempty"`
	Password        string `gorethink:"password"`
	PermissionLevel int    `gorethink:"permission"`
}

var sessionStore *rethinkstore.RethinkStore

func getUserTable() r.Term {
	return r.Table("users")
}

func initializeSessionStore() {
	store, err := rethinkstore.NewRethinkStore(siteConfiguration.RethinkConnection, "velvetdb", "sessions", 5, 5,
		[]byte(siteConfiguration.CookieKey))
	if err != nil {
		log.Fatalln(err.Error())
	}
	sessionStore = store
}

func addUser(user User) string {
	resp, err := getUserTable().Insert(&user).RunWrite(dataStore.GetSession())
	if err != nil {
		fmt.Println(err)
	}
	return resp.GeneratedKeys[0]
}

func fetchUserByEmail(email string) *User {
	var user User
	c, err := getUserTable().Filter(map[string]interface{}{
		"email": email,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = c.One(&user)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return &user
}

func validateUser(email string, password string) bool {
	user := fetchUserByEmail(email)
	if user == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func registerUser(email string, password string) {
	user := fetchUserByEmail(email)
	if user == nil {
		password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println(err)
		} else {
			addUser(User{
				Email:    email,
				Password: string(password),
			})
		}
	}
}

func isLoggedIn(r *http.Request) (string, bool) {
	storeSession, _ := sessionStore.Get(r, "usersession")
	email, found := storeSession.Values["username"].(string)
	if found {
		return email, true
	}
	return "", false
}

func saveRegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	registerUser(r.FormValue("email"), r.FormValue("password"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func saveLoginUserHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	valid := validateUser(email, password)

	if valid {
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.
		storeSession, _ := sessionStore.Get(r, "usersession")
		// Set some session values.
		storeSession.Values["username"] = email
		// Save it before we write to the response/return from the handler.
		storeSession.Save(r, w)

		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func saveLogoutUserHandler(w http.ResponseWriter, r *http.Request) {
	storeSession, _ := sessionStore.Get(r, "usersession")
	delete(storeSession.Values, "username")
	storeSession.Options.MaxAge = -1
	_ = storeSession.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "register", nil)
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login", nil)
}
