package main

import (
	"crypto/md5"
    "strconv"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/http"
	"strings"
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
	err := db.conn.Select(&cards, "SELECT id, name, rules, cmc, mana_cost, power, toughness, loyalty, type, color FROM cards ORDER BY name ASC LIMIT 100 OFFSET $1", q.Page * 100)

	if err != nil {
		return cards, err
	}

	return cards, nil
}

func (db *Database) FetchCard(id string) (Card, error) {
	var card Card

	err := db.conn.Get(&card, "SELECT name, id, cmc, mana_cost FROM cards WHERE id=$1", id)

	if err != nil {
		return card, err
	}

	return card, nil
}

func NewQuery(req *http.Request) (Query, error) {
	q := Query{}
    pagenum := req.URL.Query().Get("page")

    if pagenum == "" {
            pagenum = "0"
    }

    page, err := strconv.Atoi(pagenum)

    if err != nil {
            return q, err
    }

    q.Page = page

	return q, nil
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
	return Card{
		Name:          c.Name,
		Id:            makeId(c),
		Text:          c.Text,
		Color:         strings.Join(c.Colors, ","),
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		Type:          c.Type,
		ManaCost:      c.ManaCost,
		ConvertedCost: int(c.ConvertedCost),
	}
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
	tx := db.conn.MustBegin()

	for _, card := range TransformCollection(collection) {
		// Not sure how to handle failure here
		_, err := tx.NamedExec("INSERT INTO cards (id, name, cmc, mana_cost, rules, loyalty, power, toughness, type, color) VALUES (:id, :name, :cmc, :mana_cost, :rules, :loyalty, :power, :toughness, :type, :color)", &card)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}
