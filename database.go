package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

func GetDatabaseURL() (string, error) {
	user := os.Getenv("DATABASE_USER")
	pass := os.Getenv("DATABASE_PASSWORD")

	if user == "" || pass == "" {
		return "", fmt.Errorf("DATABASE_USER and DATABASE_PASSWORD need to be set")
	}

	return fmt.Sprintf("postgres://%s:%s@localhost/deckbrew?sslmode=disable", user, pass), nil
}

func GetDatabase() (*sql.DB, error) {
	u, err := GetDatabaseURL()

	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", u)

	if err != nil {
		return db, err
	}

	if db.Ping() != nil {
		return db, db.Ping()
	}
	return db, nil
}

func FetchSet(db *sql.DB, id string) (Set, error) {
	var set Set
	row := db.QueryRow("SELECT id,name,border,type FROM sets WHERE id = $1", id)
	err := row.Scan(&set.Id, &set.Name, &set.Border, &set.Type)
	set.Fill()
	return set, err
}

func FetchSets(db *sql.DB) ([]Set, error) {
	sets := []Set{}
	rows, err := db.Query("SELECT id,name,border,type FROM sets ORDER BY name")
	if err != nil {
		return sets, err
	}
	defer rows.Close()
	for rows.Next() {
		var set Set
		if err := rows.Scan(&set.Id, &set.Name, &set.Border, &set.Type); err != nil {
			return sets, err
		}
		set.Fill()
		sets = append(sets, set)
	}
	return sets, nil
}

func FetchTerms(db *sql.DB, term string) ([]string, error) {
	result := []string{}

	rows, err := db.Query("select distinct unnest(" + term + ") as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var term string

		if err := rows.Scan(&term); err != nil {
			return result, err
		}

		result = append(result, term)
	}

	return result, rows.Err()
}

func FetchCards(db *sql.DB, cond Condition) ([]Card, error) {
	cards := []Card{}

	query := Select("record").From("cards").Where(cond).Limit(100).OrderBy("name", true)

	ql, items, err := query.ToSql()

	if err != nil {
		return cards, err
	}

	log.Println(ql, items)

	rows, err := db.Query(ql, items...)

	if err != nil {
		return cards, err
	}

	defer rows.Close()
	for rows.Next() {
		var blob []byte
		var card Card

		if err := rows.Scan(&blob); err != nil {
			return cards, err
		}

		err = json.Unmarshal(blob, &card)

		if err != nil {
			return cards, err
		}

		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return cards, err
	}
	return cards, nil
}

func FetchCard(db *sql.DB, id string) (Card, error) {
	var blob []byte
	var card Card

	err := db.QueryRow("SELECT record FROM cards WHERE id = $1", id).Scan(&blob)

	if err == sql.ErrNoRows {
		return card, fmt.Errorf("No card with ID %s", id)
	}
	if err != nil {
		return card, err
	}

	err = json.Unmarshal(blob, &card)

	if err != nil {
		return card, err
	}

	return card, nil
}
