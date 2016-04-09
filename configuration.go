package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
)

type Configuration struct {
	RethinkConnection    string `json:"rethinkConnection"`
	ChallongeApiKey      string `json:"challongeApiKey"`
	ChallongeDevUsername string `json:"challongeDevUsername"`
	CookieKey            string `json:"cookieKey"`
}

func getConfiguration() *Configuration {
	var configuration Configuration
	file, err := os.Open("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			err := envconfig.Process("VELVETDB", &configuration)
			fmt.Println(configuration)
			if err != nil {
				log.Fatal(err.Error())
			}
		} else {
			fmt.Println("error:", err)
		}
	} else {
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&configuration)
		if err != nil {
			fmt.Println("error:", err)
		}
	}
	return &configuration
}
