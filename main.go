package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/oxtoacart/bpool"
	r "gopkg.in/dancannon/gorethink.v2"
)

type Page struct {
	User             *User
	PermissionLevels PermissionLevels
	Data             interface{}
}

type key int

const UserKey key = 0

var siteConfiguration *Configuration
var templates map[string]*template.Template
var bufpool *bpool.BufferPool
var dataStore *DataStore

func parseTemplates() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	layouts, err := filepath.Glob("layouts/*.html")
	if err != nil {
		log.Fatal(err)
	}

	includes, err := filepath.Glob("includes/*.html")
	if err != nil {
		log.Fatal(err)
	}

	for _, layout := range layouts {
		files := append(includes, layout)
		templates[filepath.Base(layout)] = template.Must(template.ParseFiles(files...))
	}
}

func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	tmpl, ok := templates[templateName+".html"]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	buf := bufpool.Get()
	defer bufpool.Put(buf)

	var user *User
	if email, ok := isLoggedIn(r); ok {
		user = fetchUserByEmail(email)
	}

	page := Page{
		User:             user,
		PermissionLevels: getPermissionLevels(),
		Data:             data,
	}

	err := tmpl.ExecuteTemplate(buf, "base", page)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "home", nil)
}

func initializeTables() {
	r.DBCreate("velvetdb").Run(dataStore.GetSession())
	r.TableCreate("users").Run(dataStore.GetSession())
	r.TableCreate("players").Run(dataStore.GetSession())
	r.TableCreate("gametypes").Run(dataStore.GetSession())
	r.TableCreate("matches").Run(dataStore.GetSession())
	r.TableCreate("tournaments").Run(dataStore.GetSession())
	r.TableCreate("tournamentresults").Run(dataStore.GetSession())
	r.TableCreate("sessions").Run(dataStore.GetSession())
}

func isAdminMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := isLoggedIn(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	})
}

func hasPermissionMiddleware(next func(http.ResponseWriter, *http.Request), permission int) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := isLoggedIn(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		u := fetchUserByEmail(email)
		if u == nil {
			http.NotFound(w, r)
			return
		}
		if !u.HasPermission(permission) {
			http.NotFound(w, r)
			return
		}
		next(w, r)
	})
}

func main() {
	siteConfiguration = getConfiguration()
	session, err := r.Connect(r.ConnectOpts{
		Address:  siteConfiguration.RethinkConnection,
		Database: "velvetdb",
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
	dataStore = &DataStore{session: session}

	initializeTables()
	initializeSessionStore()

	bufpool = bpool.NewBufferPool(64)

	parseTemplates()

	r := mux.NewRouter()

	// Serve files from the assets directory
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/",
		http.FileServer(http.Dir("assets/"))))

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/editplayer/{playerNick:[-a-zA-Z0-9]+}", isAdminMiddleware(editPlayerHandler))
	r.HandleFunc("/players", playersHandler)
	r.HandleFunc("/player/{playerNick:[-a-zA-Z0-9]+}", playerViewHandler)
	r.HandleFunc("/addplayer", isAdminMiddleware(addPlayerHandler))
	r.HandleFunc("/addgametype", isAdminMiddleware(addGameTypeHandler))
	r.HandleFunc("/addmatch", isAdminMiddleware(addMatchHandler))
	r.HandleFunc("/edit/match/{match:[-a-zA-Z0-9]+}", isAdminMiddleware(editMatchHandler))
	r.HandleFunc("/save/match/{match:[-a-zA-Z0-9]+}", isAdminMiddleware(saveEditMatchHandler))
	r.HandleFunc("/addtournament", isAdminMiddleware(addTournamentHandler))
	r.HandleFunc("/addpool/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(addPoolHandler))
	r.HandleFunc("/save/addpool", isAdminMiddleware(savePoolHandler))
	r.HandleFunc("/tournaments", viewTournamentsHandler)
	r.HandleFunc("/tournaments/{gametype}", viewTournamentsHandler)
	r.HandleFunc("/save/addmatch", isAdminMiddleware(saveMatchHandler))
	r.HandleFunc("/save/addplayer", isAdminMiddleware(savePlayerHandler))
	r.HandleFunc("/save/editplayer/{playerNick:[-a-zA-Z0-9]+}", isAdminMiddleware(saveEditPlayerHandler))
	r.HandleFunc("/save/addgametype", isAdminMiddleware(saveGameTypeHandler))
	r.HandleFunc("/save/addtournament", isAdminMiddleware(saveTournamentHandler))
	r.HandleFunc("/edit/tournament/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(editTournamentHandler))
	r.HandleFunc("/save/tournament/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(saveEditTournamentHandler))
	r.HandleFunc("/save/addtournamentmatch/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(saveTournamentMatchesHandler))
	r.HandleFunc("/tournament/addmatches/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(addTournamentMatchesHandler))
	r.HandleFunc("/tournament/{tournament:[-a-zA-Z0-9]+}", viewTournamentHandler)
	r.HandleFunc("/tournament/delete/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(deleteTournamentHandler))
	r.HandleFunc("/save/tournament/delete/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(saveDeleteTournamentHandler))

	// Tournament results
	r.HandleFunc("/tournamentresult/edit/{result:[-a-zA-Z0-9]+}", isAdminMiddleware(editTournamentResultHandler))
	r.HandleFunc("/save/tournamentresult/edit/{result:[-a-zA-Z0-9]+}", isAdminMiddleware(saveEditTournamentResultHandler))

	// Merge players
	r.HandleFunc("/players/merge", isAdminMiddleware(mergePlayersHandler))
	r.HandleFunc("/save/merge/players", isAdminMiddleware(saveMergePlayersHandler))

	// First run
	r.HandleFunc("/firstrun", firstRunHandler)
	r.HandleFunc("/firstrun/save", saveFirstRunHandler)

	// Faceoff
	r.HandleFunc("/faceoff", faceoffHandler)

	// Rankings
	r.HandleFunc("/rankings", rankingsHandler)
	r.HandleFunc("/rankings/{gametype}", rankingsHandler)

	// auth
	r.HandleFunc("/users", hasPermissionMiddleware(userListHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/profile", isAdminMiddleware(userProfileHandler))
	r.HandleFunc("/adduser", hasPermissionMiddleware(registerUserHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/login", loginUserHandler)
	r.HandleFunc("/save/login", saveLoginUserHandler)
	r.HandleFunc("/save/logout", saveLogoutUserHandler)
	r.HandleFunc("/save/adduser", hasPermissionMiddleware(saveRegisterUserHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/save/changepassword", isAdminMiddleware(saveChangePasswordHandler))

	// API
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Methods("OPTIONS").HandlerFunc(handleAPIPreflight)
	api.HandleFunc("/gametypes", handleAPIGameTypes)
	api.HandleFunc("/players", handleAPIPlayers)
	api.HandleFunc("/players/search", handleAPIPlayersSearch)
	api.HandleFunc("/players/{id:[-a-zA-Z0-9]+}", handleAPIPlayer)
	api.HandleFunc("/players/{id:[-a-zA-Z0-9]+}/tournamentresults", handleAPIPlayerTournamentResults)
	api.HandleFunc("/players/{id:[-a-zA-Z0-9]+}/matches", handleAPIPlayerMatches)
	api.HandleFunc("/players/{p1:[-a-zA-Z0-9]+}/{p2:[-a-zA-Z0-9]+}/matches", handleAPIFaceoff)

	fmt.Println("We're up and running!")

	http.ListenAndServe(":3000", r)
}
