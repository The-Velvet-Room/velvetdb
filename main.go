package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
	"github.com/oxtoacart/bpool"
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

func viewHandler(w http.ResponseWriter, r *http.Request) {
	players := fetchPlayers()
	playerDict := make(map[string]Player)
	rankDict := make(map[string]*EloDict)

	for _, p := range players {
		playerDict[p.ID] = p
	}

	c, err := getGameTable().OrderBy("date").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	e := &Elo{k: 32}

	var gameResult Game
	for c.Next(&gameResult) {
		if _, ok := rankDict[gameResult.Player1]; !ok {
			rankDict[gameResult.Player1] = NewEloDict(playerDict[gameResult.Player1])
		}
		if _, ok := rankDict[gameResult.Player2]; !ok {
			rankDict[gameResult.Player2] = NewEloDict(playerDict[gameResult.Player2])
		}
		expectedScore1 := e.getExpected(rankDict[gameResult.Player1].Rank, rankDict[gameResult.Player2].Rank)
		expectedScore2 := e.getExpected(rankDict[gameResult.Player2].Rank, rankDict[gameResult.Player1].Rank)

		player1results := float64(gameResult.Player1score) / float64(gameResult.Player1score+gameResult.Player2score)

		rankDict[gameResult.Player1].Rank = e.updateRating(expectedScore1, player1results, rankDict[gameResult.Player1].Rank)
		rankDict[gameResult.Player2].Rank = e.updateRating(expectedScore2, 1-player1results, rankDict[gameResult.Player2].Rank)
	}

	ranks := make([]*EloDict, len(rankDict))
	idx := 0
	for _, v := range rankDict {
		ranks[idx] = v
		idx++
	}

	sort.Sort(ByRank(ranks))

	renderTemplate(w, r, "view", ranks)
}

func initializeTables() {
	r.DBCreate("velvetdb").Run(dataStore.GetSession())
	r.TableCreate("users").Run(dataStore.GetSession())
	r.TableCreate("players").Run(dataStore.GetSession())
	r.TableCreate("gametypes").Run(dataStore.GetSession())
	r.TableCreate("games").Run(dataStore.GetSession())
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

	r.HandleFunc("/", viewHandler)
	r.HandleFunc("/editplayer/{playerNick:[a-zA-Z0-9]+}", isAdminMiddleware(editPlayerHandler))
	r.HandleFunc("/player/{playerNick:[a-zA-Z0-9]+}", playerViewHandler)
	r.HandleFunc("/addplayer", isAdminMiddleware(addPlayerHandler))
	r.HandleFunc("/addgametype", isAdminMiddleware(addGameTypeHandler))
	r.HandleFunc("/addgame", isAdminMiddleware(addGameHandler))
	r.HandleFunc("/addtournament", isAdminMiddleware(addTournamentHandler))
	r.HandleFunc("/tournaments", viewTournamentsHandler)
	r.HandleFunc("/tournaments/{gametype}", viewTournamentsHandler)
	r.HandleFunc("/save/addgame", isAdminMiddleware(saveGameHandler))
	r.HandleFunc("/save/addplayer", isAdminMiddleware(savePlayerHandler))
	r.HandleFunc("/save/editplayer/{playerNick:[a-zA-Z0-9]+}", isAdminMiddleware(saveEditPlayerHandler))
	r.HandleFunc("/save/addgametype", isAdminMiddleware(saveGameTypeHandler))
	r.HandleFunc("/save/addtournament", isAdminMiddleware(saveTournamentHandler))
	r.HandleFunc("/save/addtournamentmatch/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(saveTournamentMatchesHandler))
	r.HandleFunc("/tournament/addmatches/{tournament:[-a-zA-Z0-9]+}", isAdminMiddleware(addTournamentMatchesHandler))
	r.HandleFunc("/tournament/{tournament:[-a-zA-Z0-9]+}", viewTournamentHandler)

	// First run
	r.HandleFunc("/firstrun", firstRunHandler)
	r.HandleFunc("/firstrun/save", saveFirstRunHandler)

	// Faceoff
	r.HandleFunc("/faceoff", faceoffHandler)

	// auth
	r.HandleFunc("/users", hasPermissionMiddleware(userListHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/profile", isAdminMiddleware(userProfileHandler))
	r.HandleFunc("/adduser", hasPermissionMiddleware(registerUserHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/login", loginUserHandler)
	r.HandleFunc("/save/login", saveLoginUserHandler)
	r.HandleFunc("/save/logout", saveLogoutUserHandler)
	r.HandleFunc("/save/adduser", hasPermissionMiddleware(saveRegisterUserHandler, getPermissionLevels().CanModifyUsers))
	r.HandleFunc("/save/changepassword", isAdminMiddleware(saveChangePasswordHandler))

	fmt.Println("We're up and running!")

	http.ListenAndServe(":3000", r)
}
