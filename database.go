package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"net/http"
)

type Query struct {
	PageSize int
	Page     int
	Types    map[string]bool
}

type Database struct {
	conn *sql.DB
}

func (db *Database) FetchCard(id string) (Card, error) {
	var name string
	err := db.conn.QueryRow("SELECT name FROM cards WHERE id=$1", id).Scan(&name)
	switch {
	case err == sql.ErrNoRows:
		return Card{}, fmt.Errorf("No card with ID %s could be found", id)
	case err != nil:
		return Card{}, err
	default:
		return Card{Name: name, Id: id}, nil
	}
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

func NewConnection(url string) (Database, error) {
	conn, err := sql.Open("postgres", "postgres://localhost/deckbrew?sslmode=disable")

	if err != nil {
		return Database{}, err
	}

	return Database{conn: conn}, nil
}

func (db *Database) FetchCards(q Query) []Card {
	return []Card{}
}

// Given an array of cards, load them into the database
func (db *Database) Load(cards []Card) error {
	tx, err := db.conn.Begin()

	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO cards (id, name) VALUES ($1, $2)")
	defer stmt.Close()

	if err != nil {
		return err
	}

	for _, card := range cards {
		// Not sure how to handle failure here
		_, err = stmt.Exec(card.Id, card.Name)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}
