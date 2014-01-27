package main

import (
        "fmt"
)

type Database struct {
	Cards []Card `json:"cards"`
}

func (db *Database) FetchCard(id string) (Card, error) {
        for _, card := range db.Cards {
                if card.Id == id {
                        return card, nil
                }
        }
        return Card{}, fmt.Errorf("No card with ID %s could be found", id)
}


func (db *Database) FetchCards(pageSize int, page int) []Card {
        return db.Cards[:pageSize]
}
