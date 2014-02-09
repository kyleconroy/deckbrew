package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

type Person struct {
	Name  string
	Phone string
}

// FIXME: Add TX support
func LoadMongo(session *mgo.Session, collection MTGCollection) error {

	_, cards, _ := TransformCollection(collection)

	cardCollection := session.DB("deckbrew").C("cards")

	for _, c := range cards {
		err := cardCollection.Insert(&c)

		if err != nil {
			return err
		}
	}

	return nil
}

func CreateIndexes(session *mgo.Session) error {
	cardCollection := session.DB("deckbrew").C("cards")

	index := mgo.Index{
		Key:      []string{"name"},
		Unique:   true,
		DropDups:   true,
	}

    err := cardCollection.EnsureIndex(index)

    if err != nil {
            return err
}

	index = mgo.Index{
		Key:      []string{"editions.multiverseid"},
	}

	return cardCollection.EnsureIndex(index)
}

func NewLoad(path string) error {
	session, err := mgo.Dial("localhost:27017")

	if err != nil {
		return err
	}

	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	if err != nil {
		return err
	}

	err = session.DB("deckbrew").DropDatabase()

	if err != nil {
		return err
	}

	collection, err := LoadCollection(path)

	if err != nil {
		return err
	}

	err = LoadMongo(session, collection)

	if err != nil {
		return err
	}

	err = CreateIndexes(session)

	if err != nil {
		return err
	}

	var card Card

	cardCollection := session.DB("deckbrew").C("cards")

	err = cardCollection.Find(bson.M{"_id": "a86e8832461ee5e9cfb79b8584989f78"}).One(&card)

	if err != nil {
		return err
	}

	log.Printf("%+v", card)
	return nil
}
