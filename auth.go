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
	PermissionLevel   int    `gorethink:"permission"`
}

type PermissionLevels struct {
	CanModifyUsers int
}

func (u User) HasPermission(p int) bool {
	return u.PermissionLevel&p != 0
}

var sessionStore *rethinkstore.RethinkStore

func getUserTable() r.Term {
	return r.Table("users")
}

func getPermissionLevels() PermissionLevels {
	return PermissionLevels{
		CanModifyUsers: 1,
	}
}

func getMaxPermissionLevel() int {
	return 1
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

func updateUserPassword(email string, newPassword string) error {
	p, err := generatePassword(newPassword)
	if err != nil {
		return err
	}

	_, err = getUserTable().Filter(map[string]interface{}{
		"email": email,
	}).Update(map[string]interface{}{
		"password": p,
	}).RunWrite(dataStore.GetSession())

	return err
}

func fetchUserByEmail(email string) *User {
	c, err := getUserTable().Filter(map[string]interface{}{
		"email": email,
	}).Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var user User
	err = c.One(&user)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return &user
}

func fetchUsers() []User {
	c, err := getUserTable().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}
	var u []User
	err = c.All(&u)
	if err != nil {
		fmt.Println(err)
	}
	return u
}

func generatePassword(password string) (string, error) {
	p, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func validateUser(email string, password string) bool {
	user := fetchUserByEmail(email)
	if user == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func registerUser(email string, password string, permissionLevel int) {
	user := fetchUserByEmail(email)
	if user == nil {
		p, err := generatePassword(password)
		if err != nil {
			fmt.Println(err)
		} else {
			addUser(User{
				Email:           email,
				Password:        p,
				PermissionLevel: permissionLevel,
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
	e := r.FormValue("email")
	p := r.FormValue("password")
	pa := r.FormValue("passwordAgain")
	cm := r.FormValue("canModifyUsers")
	data := struct{ Message string }{"User added."}
	if p != pa {
		data.Message = "Passwords didn't match."
		renderTemplate(w, r, "register", data)
		return
	}
	if fetchUserByEmail(e) != nil {
		data.Message = "User already exists."
		renderTemplate(w, r, "register", data)
		return
	}
	pl := 0
	if cm != "" {
		pl = getPermissionLevels().CanModifyUsers
	}
	registerUser(e, p, pl)
	renderTemplate(w, r, "register", data)
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
	_ = storeSession.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func saveChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	currentPass := r.FormValue("currentPass")
	newPass := r.FormValue("newPass")
	newPassAgain := r.FormValue("newPassAgain")
	data := struct{ Message string }{"Password changed successfully."}
	if newPass != newPassAgain {
		data.Message = "The new passwords didn't match."
		renderTemplate(w, r, "userProfile", data)
		return
	}
	email, _ := isLoggedIn(r)
	if validateUser(email, currentPass) {
		err := updateUserPassword(email, newPass)
		if err != nil {
			data.Message = "An error occurred. Please contact an administrator."
			fmt.Println(err)
		}
	} else {
		data.Message = "The password did not match the password for this account."
	}

	renderTemplate(w, r, "userProfile", data)
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "register", nil)
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login", nil)
}

func userProfileHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "userProfile", nil)
}

func userListHandler(w http.ResponseWriter, r *http.Request) {
	u := fetchUsers()
	data := struct {
		Users []User
	}{
		u,
	}
	renderTemplate(w, r, "userList", data)
}
