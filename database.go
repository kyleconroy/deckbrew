package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strings"
)

func getDatabase() (*sql.DB, error) {
	url := os.Getenv("DECKBREW_DATABASE")

	if url == "" {
		return nil, fmt.Errorf("connection requires DECKBREW_DATABASE environment variable")
	}

	db, err := sql.Open("postgres", url)

	if err != nil {
		return db, err
	}

	return db, nil
}

func FetchSet(db *sql.DB, id string) (Set, error) {
	var set Set
	row := db.QueryRow("SELECT id,name,border,type FROM sets WHERE id = $1", id)
	err := row.Scan(&set.Id, &set.Name, &set.Border, &set.Type)
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

func scanCards(rows *sql.Rows) ([]Card, error) {
	cards := []Card{}

	defer rows.Close()

	for rows.Next() {
		var blob []byte
		var card Card

		if err := rows.Scan(&blob); err != nil {
			return cards, err
		}

		err := json.Unmarshal(blob, &card)

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

func FetchTypeahead(db *sql.DB, search string) ([]Card, error) {
	if strings.ContainsAny(search, "%_") {
		return []Card{}, fmt.Errorf("Search string can't contain '%%' or '_'")
	}

	rows, err := db.Query("SELECT record FROM cards WHERE name ILIKE $1 ORDER BY name LIMIT 10", search+"%")

	if err != nil {
		return []Card{}, err
	}

	return scanCards(rows)
}

func FetchCards(db *sql.DB, cond Condition, page int) ([]Card, error) {
	query := Select("record").From("cards").Where(cond).OrderBy("name", true)
	limit := query.Limit(100).Offset(page * 100)

	ql, items, err := limit.ToSql()

	if err != nil {
		return []Card{}, err
	}

	log.Println(ql, items)

	rows, err := db.Query(ql, items...)

	if err != nil {
		return []Card{}, err
	}

	return scanCards(rows)
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
	return card, json.Unmarshal(blob, &card)
}
