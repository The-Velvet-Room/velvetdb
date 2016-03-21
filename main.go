package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/oxtoacart/bpool"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
		session := dataStore.GetSession()
		defer session.Close()
		user = fetchUserByEmail(session, email)
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

func addGameHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()
	// get players
	players := fetchPlayers(session)

	gameTypes := fetchGameTypes(session)

	data := struct {
		Players   []Player
		GameTypes []GameType
	}{
		players,
		gameTypes,
	}

	renderTemplate(w, r, "addGame", data)
}

func testBulkHandler(w http.ResponseWriter, r *http.Request) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	gc := getGameCollection(session)

	bulk := gc.Bulk()
	player1 := bson.ObjectIdHex("56e20beedbd03b5fcc642772")
	player2 := bson.ObjectIdHex("56e20c11dbd03b5fcc642778")
	gametype := bson.ObjectIdHex("56e0ce10903e6dc358d25d8c")

	for index := 0; index < 5000; index++ {
		p1score := rand.Intn(3)
		p2score := rand.Intn(2)
		if p1score != 2 {
			p2score = 2
		}
		bulk.Insert(Game{
			ID:           bson.NewObjectId(),
			GameType:     gametype,
			Player1:      player1,
			Player2:      player2,
			Player1score: p1score,
			Player2score: p2score,
		})
	}

	result, err := bulk.Run()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("modified:", result.Modified)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	c := getPlayerCollection(session)
	players := c.Find(nil).Iter()

	playerDict := make(map[bson.ObjectId]*EloDict)

	var result Player
	for players.Next(&result) {
		playerDict[result.ID] = &EloDict{Rank: 1000, Name: result.Nickname, ID: result.ID.Hex()}
	}

	gameSession := getGameCollection(session)
	games := gameSession.Find(nil).Sort("date").Iter()
	e := &Elo{k: 32}

	var gameResult Game
	for games.Next(&gameResult) {
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
	session, err := mgo.Dial(siteConfiguration.MongoConnection)
	if err != nil {
		panic(err)
	}

	bufpool = bpool.NewBufferPool(64)
	dataStore = &DataStore{session: session}
	parseTemplates()

	r := mux.NewRouter()

	// Serve files from the assets directory
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/",
		http.FileServer(http.Dir("assets/"))))

	r.HandleFunc("/", viewHandler)
	r.HandleFunc("/player/{player:[a-zA-Z0-9]+}", playerViewHandler)
	r.HandleFunc("/addgametype", isAdminMiddleware(addGameTypeHandler))
	r.HandleFunc("/addgame", isAdminMiddleware(addGameHandler))
	r.HandleFunc("/addtournament", isAdminMiddleware(addTournamentHandler))
	r.HandleFunc("/save/addgametype", isAdminMiddleware(saveGameTypeHandler))
	r.HandleFunc("/save/addtournament", isAdminMiddleware(saveTournamentHandler))
	r.HandleFunc("/save/addtournamentmatch/{tournament:[a-zA-Z0-9]+}", isAdminMiddleware(saveTournamentMatchHandler))
	r.HandleFunc("/tournament/{tournament:[a-zA-Z0-9]+}", viewTournamentHandler)

	r.HandleFunc("/testbulk", testBulkHandler)

	// auth
	r.HandleFunc("/register", isAdminMiddleware(registerUserHandler))
	r.HandleFunc("/login", loginUserHandler)
	r.HandleFunc("/save/login", saveLoginUserHandler)
	r.HandleFunc("/save/logout", saveLogoutUserHandler)
	r.HandleFunc("/save/register", isAdminMiddleware(saveRegisterUserHandler))

	fmt.Println("We're up and running!")

	http.ListenAndServe(":3000", r)
}
