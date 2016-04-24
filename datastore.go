package main

import (
	"log"

	r "gopkg.in/dancannon/gorethink.v2"
)

type DataStore struct {
	session *r.Session
}

func (ds *DataStore) GetSession() *r.Session {
	return ds.session
}

func (ds *DataStore) GetID() string {
	c, err := r.UUID().Run(dataStore.GetSession())
	defer c.Close()
	if err != nil {
		log.Fatalln(err)
	}
	var uuid string
	err = c.One(&uuid)
	if err != nil {
		log.Fatalln(err)
	}
	return uuid
}
