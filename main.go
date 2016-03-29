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
	User *User
	Data interface{}
}

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

	email, ok := isLoggedIn(r)
	var user *User
	if ok {
		user = fetchUserByEmail(email)
	} else {
		user = nil
	}

	page := Page{
		User: user,
		Data: data,
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

	playerDict := make(map[string]*EloDict)

	for _, p := range players {
		playerDict[p.ID] = &EloDict{Rank: 1000, Player: p}
	}

	c, err := getGameTable().OrderBy("date").Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		fmt.Println(err)
	}

	e := &Elo{k: 32}

	var gameResult Game
	for c.Next(&gameResult) {
		expectedScore1 := e.getExpected(playerDict[gameResult.Player1].Rank, playerDict[gameResult.Player2].Rank)
		expectedScore2 := e.getExpected(playerDict[gameResult.Player2].Rank, playerDict[gameResult.Player1].Rank)

		player1results := float64(gameResult.Player1score) / float64(gameResult.Player1score+gameResult.Player2score)

		playerDict[gameResult.Player1].Rank = e.updateRating(expectedScore1, player1results, playerDict[gameResult.Player1].Rank)
		playerDict[gameResult.Player2].Rank = e.updateRating(expectedScore2, 1-player1results, playerDict[gameResult.Player2].Rank)
	}

	ranks := []*EloDict{}
	for _, v := range playerDict {
		ranks = append(ranks, v)
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
	r.TableCreate("sessions").Run(dataStore.GetSession())
}

func isAdminMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := isLoggedIn(r)
		if ok {
			next(w, r)
		} else {
			http.NotFound(w, r)
		}
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
	r.HandleFunc("/player/{playerNick:[a-zA-Z0-9]+}", playerViewHandler)
	r.HandleFunc("/addplayer", isAdminMiddleware(addPlayerHandler))
	r.HandleFunc("/addgametype", isAdminMiddleware(addGameTypeHandler))
	r.HandleFunc("/addgame", isAdminMiddleware(addGameHandler))
	r.HandleFunc("/addtournament", isAdminMiddleware(addTournamentHandler))
	r.HandleFunc("/save/addgame", isAdminMiddleware(saveGameHandler))
	r.HandleFunc("/save/addplayer", isAdminMiddleware(savePlayerHandler))
	r.HandleFunc("/save/addgametype", isAdminMiddleware(saveGameTypeHandler))
	r.HandleFunc("/save/addtournament", isAdminMiddleware(saveTournamentHandler))
	r.HandleFunc("/save/addtournamentmatch/{tournament:[a-zA-Z0-9]+}", isAdminMiddleware(saveTournamentMatchHandler))
	r.HandleFunc("/tournament/{tournament:[a-zA-Z0-9]+}", viewTournamentHandler)

	// First run
	r.HandleFunc("/firstrun", firstRunHandler)
	r.HandleFunc("/firstrun/save", saveFirstRunHandler)

	// auth
	r.HandleFunc("/register", isAdminMiddleware(registerUserHandler))
	r.HandleFunc("/login", loginUserHandler)
	r.HandleFunc("/save/login", saveLoginUserHandler)
	r.HandleFunc("/save/logout", saveLogoutUserHandler)
	r.HandleFunc("/save/register", isAdminMiddleware(saveRegisterUserHandler))

	fmt.Println("We're up and running!")

	http.ListenAndServe(":3000", r)
}
