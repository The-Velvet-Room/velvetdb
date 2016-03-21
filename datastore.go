package main

import (
	"gopkg.in/mgo.v2"
)

type DataStore struct {
	session *mgo.Session
}

func (ds *DataStore) GetSession() *mgo.Session {
	return ds.session.Copy()
}
