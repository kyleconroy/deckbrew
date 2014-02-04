package main

import (
	"crypto/md5"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/http"
	"strconv"
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

func (db *Database) FetchEditions(id string) ([]Card, error) {
	editions := []Edition{}
	cards := []Card{}

	err := db.conn.Select(&editions, "SELECT * FROM editions WHERE id=$1 ORDER BY id ASC", id)

	if err != nil {
		return cards, err
	}

	for _, ed := range editions {
		ed.Fill()

		var card Card

		err = db.conn.Get(&card, "SELECT * FROM cards WHERE id=$1", ed.CardId)

		if err != nil {
			continue
		}

		card.Fill()
		card.Editions = append(card.Editions, ed)
		cards = append(cards, card)
	}

	return cards, nil
}

func (db *Database) FetchSets() ([]Set, error) {
	sets := []Set{}
	err := db.conn.Select(&sets, "SELECT * FROM sets ORDER BY name ASC")

	if err != nil {
		return sets, err
	}

	for i, _ := range sets {
		sets[i].Fill()
	}

	return sets, nil
}

func (db *Database) FetchSet(id string) (Set, error) {
	var set Set

	err := db.conn.Get(&set, "SELECT * FROM sets WHERE id=$1", id)

	if err != nil {
		return set, err
	}

	set.Fill()

	return set, nil
}

func (db *Database) FetchCards(q Query) ([]Card, error) {
	cards := []Card{}
	err := db.conn.Select(&cards, "SELECT * FROM cards ORDER BY name ASC LIMIT 100 OFFSET $1", q.Page*100)

	if err != nil {
		return cards, err
	}

	for i, _ := range cards {
		cards[i].Fill()

		err = db.conn.Select(&cards[i].Editions, "SELECT * FROM editions WHERE card_id=$1 ORDER BY id ASC", cards[i].Id)

		if err != nil {
			continue
		}

		for j, _ := range cards[i].Editions {
			cards[i].Editions[j].Fill()
		}

	}

	return cards, nil
}

func (db *Database) FetchCard(id string) (Card, error) {
	var card Card

	err := db.conn.Get(&card, "SELECT * FROM cards WHERE id=$1", id)

	if err != nil {
		return card, err
	}

	card.Fill()

	err = db.conn.Select(&card.Editions, "SELECT * FROM editions WHERE card_id=$1 ORDER BY id ASC", card.Id)

	if err != nil {
		return card, err
	}

	for j, _ := range card.Editions {
		card.Editions[j].Fill()
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

	if page < 0 {
		return q, fmt.Errorf("Page parameter must be >= 0")
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

func join(things []string) string {
	return strings.ToLower(strings.Join(things, ","))
}

func TransformEdition(s MTGSet, c MTGCard) Edition {
	return Edition{
		Set:          s.Name,
		SetId:        s.Code,
		Flavor:       c.Flavor,
		MultiverseId: c.MultiverseId,
		Watermark:    c.Watermark,
		Rarity:       c.Rarity,
		Artist:       c.Artist,
		Border:       c.Border,
		Layout:       c.Layout,
		Number:       c.Number,
		CardId:       makeId(c),
	}
}

func TransformSet(s MTGSet) Set {
	// FIXME: Add released dates
	return Set{
		Name:   s.Name,
		Id:     s.Code,
		Border: s.Border,
		Type:   s.Type,
	}
}

func TransformCard(c MTGCard) Card {
	return Card{
		Name:             c.Name,
		Id:               makeId(c),
		Text:             c.Text,
		JoinedColors:     join(c.Colors),
		JoinedTypes:      join(c.Types),
		JoinedSupertypes: join(c.Supertypes),
		JoinedSubtypes:   join(c.Subtypes),
		Power:            c.Power,
		Toughness:        c.Toughness,
		Loyalty:          c.Loyalty,
		ManaCost:         c.ManaCost,
		ConvertedCost:    int(c.ConvertedCost),
	}
}

func TransformCollection(collection MTGCollection) ([]Set, []Card, []Edition) {
	cards := []Card{}
	ids := map[string]Card{}
	editions := []Edition{}
	sets := []Set{}

	for _, set := range collection {
		sets = append(sets, TransformSet(set))

		for _, card := range set.Cards {
			newcard := TransformCard(card)
			newedition := TransformEdition(set, card)

			if _, found := ids[newcard.Id]; !found {
				ids[newcard.Id] = newcard
				cards = append(cards, newcard)
			}

			editions = append(editions, newedition)
		}
	}
	return sets, cards, editions
}

// Given an array of cards, load them into the database
func (db *Database) Load(collection MTGCollection) error {
	tx := db.conn.MustBegin()

	sets, cards, editions := TransformCollection(collection)

	for _, set := range sets {
		// Not sure how to handle failure here
		_, err := tx.NamedExec("INSERT INTO sets (id, name, border, type) VALUES (:id, :name, :border, :type)", &set)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, card := range cards {
		// Not sure how to handle failure here
		_, err := tx.NamedExec("INSERT INTO cards (id, name, cmc, mana_cost, rules, loyalty, power, toughness, types, supertypes, subtypes, colors) VALUES (:id, :name, :cmc, :mana_cost, :rules, :loyalty, :power, :toughness, :types, :supertypes, :subtypes, :colors)", &card)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	for _, edition := range editions {
		// Not sure how to handle failure here
		_, err := tx.NamedExec("INSERT INTO editions (id, card_id, set_name, watermark, rarity, border, artist, flavor, set_number, layout, set_id) VALUES (:id, :card_id, :set_name, :watermark, :rarity, :border, :artist, :flavor, :set_number, :layout, :set_id)", &edition)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}
