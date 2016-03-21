package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Game struct {
	ID                bson.ObjectId `bson:"_id,omitempty"`
	Tournament        bson.ObjectId `bson:",omitempty"`
	TournamentMatchID string
	GameType          bson.ObjectId
	Date              time.Time
	Player1           bson.ObjectId
	Player2           bson.ObjectId
	Player1score      int
	Player2score      int
}

func getGameCollection(session *mgo.Session) *mgo.Collection {
	return session.DB("test").C("games")
}

func fetchGamesForPlayer(session *mgo.Session, id bson.ObjectId) []Game {
	c := getGameCollection(session)
	var result *Game
	games := []Game{}
	gameIter := c.Find(bson.M{"$or": []bson.M{bson.M{"player1": id}, bson.M{"player2": id}}}).Iter()
	for gameIter.Next(&result) {
		games = append(games, *result)
	}
	return games
}

func fetchGamesForTournament(session *mgo.Session, id bson.ObjectId) []Game {
	c := getGameCollection(session)
	var result *Game
	games := []Game{}
	gameIter := c.Find(bson.M{"tournament": id}).Sort("date").Iter()
	for gameIter.Next(&result) {
		games = append(games, *result)
	}
	return games
}

func addGame(session *mgo.Session, game Game) {
	gc := getGameCollection(session)
	err := gc.Insert(game)
	if err != nil {
		fmt.Println(err)
	}
}

func saveGameHandler(w http.ResponseWriter, r *http.Request) {
	session := dataStore.GetSession()
	defer session.Close()

	player1 := r.FormValue("player1")
	player2 := r.FormValue("player2")
	player1score, _ := strconv.Atoi(r.FormValue("player1score"))
	player2score, _ := strconv.Atoi(r.FormValue("player2score"))
	gameType := r.FormValue("gametype")
	if player1 == player2 || !bson.IsObjectIdHex(player1) ||
		!bson.IsObjectIdHex(player2) ||
		!bson.IsObjectIdHex(gameType) {
		http.Redirect(w, r, "/", http.StatusBadRequest)
	}

	c := getGameCollection(session)
	saveErr := c.Insert(&Game{
		Player1:      bson.ObjectIdHex(player1),
		Player2:      bson.ObjectIdHex(player2),
		GameType:     bson.ObjectIdHex(gameType),
		Player1score: player1score,
		Player2score: player2score,
		Date:         time.Now(),
	})
	if saveErr != nil {
		fmt.Println(saveErr)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
