package main

import (
	"fmt"
	"net/http"

	"github.com/kidstuff/mongostore"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	ID              bson.ObjectId `bson:"_id,omitempty"`
	Email           string
	Player          bson.ObjectId `bson:",omitempty"`
	Password        string
	PermissionLevel int
}

func getUserCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("users")
}

func getSessionCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("sessions")
}

func addUser(session *mgo.Session, user User) bson.ObjectId {
	c := getUserCollection(session)
	if !user.ID.Valid() {
		user.ID = bson.NewObjectId()
	}
	fmt.Println("valid", user)

	saveErr := c.Insert(&user)
	if saveErr != nil {
		fmt.Println(saveErr)
	}
	return user.ID
}

func fetchUserByEmail(session *mgo.Session, email string) *User {
	c := getUserCollection(session)
	var user User
	err := c.Find(bson.M{"email": email}).One(&user)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return &user
}

func validateUser(session *mgo.Session, email string, password string) bool {
	user := fetchUserByEmail(session, email)
	if user == nil {
		fmt.Println("No user found")
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func registerUser(session *mgo.Session, email string, password string) {
	user := fetchUserByEmail(session, email)
	if user == nil {
		password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println(err)
		} else {
			addUser(session, User{
				Email:    email,
				Password: string(password),
			})
		}
	}
}

func isLoggedIn(r *http.Request) (string, bool) {
	session := dataStore.GetSession()
	defer session.Close()
	store := mongostore.NewMongoStore(getSessionCollection(session), 3600, true,
		[]byte(siteConfiguration.CookieKey))

	storeSession, _ := store.Get(r, "usersession")
	email, found := storeSession.Values["username"].(string)
	if found {
		return email, true
	}
	return "", false
}

func saveRegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	registerUser(session, r.FormValue("email"), r.FormValue("password"))
	http.Redirect(w, r, "/", http.StatusFound)
}

func saveLoginUserHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	email := r.FormValue("email")
	password := r.FormValue("password")
	valid := validateUser(session, email, password)

	if valid {
		store := mongostore.NewMongoStore(getSessionCollection(session), 3600, true,
			[]byte(siteConfiguration.CookieKey))
		// Get a session. We're ignoring the error resulted from decoding an
		// existing session: Get() always returns a session, even if empty.
		storeSession, _ := store.Get(r, "usersession")
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
	session := dataStore.GetSession()
	defer session.Close()
	store := mongostore.NewMongoStore(getSessionCollection(session), 3600, true,
		[]byte(siteConfiguration.CookieKey))
	storeSession, _ := store.Get(r, "usersession")
	delete(storeSession.Values, "username")
	_ = storeSession.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func registerUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "register", nil)
}

func loginUserHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login", nil)
}
