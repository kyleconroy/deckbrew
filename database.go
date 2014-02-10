package main

import (
	"database/sql"
	"fmt"
    "encoding/json"
    "log"
	_ "github.com/lib/pq"
    "os"
)

func FetchSets(db *sql.DB) ([]Set, error) {
	sets := []Set{}

	//err := db.conn.Select(&sets, "SELECT * FROM sets ORDER BY name ASC")
	err := db.Ping()

	if err != nil {
		return sets, err
	}

	for i, _ := range sets {
		sets[i].Fill()
	}

	return sets, nil
}

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

	//err := db.conn.Get(&set, "SELECT * FROM sets WHERE id=$1", id)
	err := db.Ping()

	if err != nil {
		return set, err
	}

	//set.Fill()

	return set, nil
}

func FetchTerms(db *sql.DB, term string) ([]string, error) {
	result := []string{}

	rows, err := db.Query("select distinct unnest(" + term + ") as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC")

	if err != nil {
		return result, err
	}

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

		card.Fill()

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return cards, err
	}

	return cards, nil
}

func FetchCard(db *sql.DB, id string) (Card, error) {
	var card Card

	//err := db.ScanCard(&card, id)
    err := db.Ping()

	if err != nil {
		return card, err
	}

	//card.Fill()

	return card, nil
}
