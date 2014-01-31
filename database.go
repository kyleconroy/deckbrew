package main

import (
	"crypto/md5"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/http"
)

type Query struct {
	PageSize int
	Page     int
	Types    map[string]bool
}

type Database struct {
	conn *sqlx.DB
}

func (db *Database) FetchCards(q Query) ([]Card, error) {
    cards := []Card{}
    err := db.conn.Select(&cards, "SELECT name, id FROM cards ORDER BY name ASC LIMIT 100")

    if err != nil {
        return cards, err
    }

	return cards, nil
}


func (db *Database) FetchCard(id string) (Card, error) {
    var card Card

	err := db.conn.Get(&card, "SELECT name, id FROM cards WHERE id=$1", id)

    if err != nil {
		return card, err
    }

    return card, nil
}

func NewQuery(req *http.Request) Query {
	q := Query{}
	q.PageSize = 100
	q.Page = 0
	q.Types = map[string]bool{
		"artifact":     true,
		"creature":     true,
		"enchantment":  true,
		"instant":      true,
		"land":         true,
		"planeswalker": true,
		"sorcerty":     true,
	}
	return q
}

func Open(url string) (Database, error) {
	conn, err := sqlx.Open("postgres", url)

	if err != nil {
		return Database{}, err
	}

	return Database{conn: conn}, nil
}

func makeId(c MTGCard) string {
	h := md5.New()
	io.WriteString(h, c.Name+c.ManaCost)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TransformCard(c MTGCard) Card {
	return Card{Name: c.Name, Id: makeId(c)}
}

func TransformCollection(collection MTGCollection) []Card {
	cards := []Card{}
	ids := map[string]Card{}
	for _, set := range collection {
		for _, card := range set.Cards {
			newcard := TransformCard(card)

			if _, found := ids[newcard.Id]; !found {
			    ids[newcard.Id] = newcard
			    cards = append(cards, newcard)
			}

		}

	}
	return cards
}

// Given an array of cards, load them into the database
func (db *Database) Load(collection MTGCollection) error {
	tx, err := db.conn.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO cards (id, name) VALUES ($1, $2)")
	defer stmt.Close()

	if err != nil {
		return err
	}

	for _, card := range TransformCollection(collection) {
		// Not sure how to handle failure here
		_, err = stmt.Exec(card.Id, card.Name)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}
