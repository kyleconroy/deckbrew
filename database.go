package main

import (
	"fmt"
	"net/http"
)

type Database struct {
	Cards []Card `json:"cards"`
}

type Query struct {
	PageSize int
	Page     int
	Types    map[string]bool
}

func (db *Database) FetchCard(id string) (Card, error) {
	for _, card := range db.Cards {
		if card.Id == id {
			return card, nil
		}
	}
	return Card{}, fmt.Errorf("No card with ID %s could be found", id)
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

func (db *Database) FetchCards(q Query) []Card {
	result := []Card{}
	for _, card := range db.Cards {
		if card.Match(q) {
			result = append(result, card)
		}
		if len(result) == q.PageSize {
			return result
		}
	}
	return result
}
