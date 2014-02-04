package main

import (
	"crypto/md5"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type Query struct {
	PageSize   int
	Page       int
	Types      []string
	Supertypes []string
	Colors     []string
	Subtypes   []string
}

func (q *Query) WhereClause() (string, []interface{}) {
	query := "WHERE "
	count := 1
	items := []interface{}{}

    pgarray := func(strs []string) string {
	    sort.Strings(strs)
	    return CreateStringArray(strs)
    }

	if len(q.Types) != 0 {
		query += fmt.Sprintf("types && $%d", count)
		count += 1
		items = append(items, pgarray(q.Types))
	}

	if len(q.Subtypes) != 0 {
		if count > 1 {
			query += " AND "
		}

		query += fmt.Sprintf("subtypes && $%d", count)
		count += 1
		items = append(items, pgarray(q.Subtypes))
	}

	if len(q.Supertypes) != 0 {
		if count > 1 {
			query += " AND "
		}

		query += fmt.Sprintf("supertypes && $%d", count)
		count += 1
		items = append(items, pgarray(q.Supertypes))
	}

	if len(q.Colors) != 0 {
		if count > 1 {
			query += " AND "
		}

		query += fmt.Sprintf("colors && $%d", count)
		count += 1
		items = append(items, pgarray(q.Colors))
	}


	query += fmt.Sprintf(" ORDER BY name ASC LIMIT 100 OFFSET $%d", count)
	items = append(items, q.PageOffset())

	return query, items
}

func (q *Query) PageOffset() int {
	return q.Page * 100
}

func extractSubtypes(args url.Values) ([]string, error) {
	return args["subtype"], nil
}

func extractColors(args url.Values) ([]string, error) {
	allowedColors := map[string]bool{
		"red":   true,
		"blue":  true,
		"green": true,
		"black": true,
		"white": true,
	}

	colors := args["color"]

	if len(colors) == 0 {
		return []string{}, nil
	}

	for _, t := range colors {
		if !allowedColors[t] {
			return colors, fmt.Errorf("The color '%s' is not recognized", t)
		}
	}

	return colors, nil
}

func extractSupertypes(args url.Values) ([]string, error) {
	allowedTypes := map[string]bool{
		"legendary": true,
		"basic":     true,
		"world":     true,
		"snow":      true,
		"ongoing":   true,
	}

	types := args["supertype"]

	if len(types) == 0 {
		return []string{}, nil
	}

	for _, t := range types {
		if !allowedTypes[t] {
			return types, fmt.Errorf("The supertype '%s' is not recognized", t)
		}
	}

	return types, nil
}

func extractTypes(args url.Values) ([]string, error) {
	allowedTypes := map[string]bool{
		"creature":     true,
		"land":         true,
		"tribal":       true,
		"phenomenon":   true,
		"summon":       true,
		"enchantment":  true,
		"sorcery":      true,
		"vanguard":     true,
		"instant":      true,
		"planeswalker": true,
		"artifact":     true,
		"plane":        true,
		"scheme":       true,
	}

	defaultTypes := []string{
		"creature", "land", "enchantment", "sorcery",
		"instant", "planeswalker", "artifact",
	}

	types := args["type"]

	if len(types) == 0 {
		return defaultTypes, nil
	}

	for _, t := range types {
		if !allowedTypes[t] {
			return types, fmt.Errorf("The type '%s' is not recognized", t)
		}
	}

	return types, nil
}

func extractPage(args url.Values) (int, error) {
	pagenum := args.Get("page")

	if pagenum == "" {
		pagenum = "0"
	}

	page, err := strconv.Atoi(pagenum)

	if err != nil {
		return 0, err
	}

	if page < 0 {
		return 0, fmt.Errorf("Page parameter must be >= 0")
	}

	return page, nil
}

func NewQuery(u *url.URL) (Query, error) {
	var err error

	args := u.Query()
	q := Query{}

	q.Page, err = extractPage(args)

	if err != nil {
		return q, err
	}

	q.Types, err = extractTypes(args)

	if err != nil {
		return q, err
	}

	q.Supertypes, err = extractSupertypes(args)

	if err != nil {
		return q, err
	}

	q.Subtypes, err = extractSubtypes(args)

	if err != nil {
		return q, err
	}

	q.Colors, err = extractColors(args)

	if err != nil {
		return q, err
	}

	return q, nil
}

type Database struct {
	conn *sqlx.DB
}

func (db *Database) ScanCard(c *Card, id string) error {
	return db.conn.Get(c, "SELECT name, id, array_to_string(types, ',') AS types, array_to_string(subtypes, ',') AS subtypes, array_to_string(supertypes, ',') AS supertypes, array_to_string(colors, ',') AS colors, mana_cost, cmc, loyalty, rules FROM cards WHERE id=$1", id)
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

		err = db.ScanCard(&card, ed.CardId)

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

	clause, items := q.WhereClause()

	err := db.conn.Select(&cards, "SELECT name, id, array_to_string(types, ',') AS types, array_to_string(subtypes, ',') AS subtypes, array_to_string(supertypes, ',') AS supertypes, array_to_string(colors, ',') AS colors, mana_cost, cmc, loyalty, rules FROM cards "+clause, items...)

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

	err := db.ScanCard(&card, id)

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

// FIXME: This function is super gross. Instead of abusing regexes,
// we should actually be parsing the search string using a lexer
// and stuff

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

func normalize(things []string) []string {
	sorted := []string{}
	for _, thing := range things {
		sorted = append(sorted, strings.ToLower(strings.Replace(thing, ",", "", -1)))
	}
	sort.Strings(sorted)
	return sorted
}

func TransformEdition(s MTGSet, c MTGCard) Edition {
	return Edition{
		Set:          s.Name,
		SetId:        s.Code,
		Flavor:       c.Flavor,
		MultiverseId: c.MultiverseId,
		Watermark:    c.Watermark,
		Rarity:       strings.ToLower(c.Rarity),
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
		Name:          c.Name,
		Id:            makeId(c),
		Text:          c.Text,
		Colors:        normalize(c.Colors),
		Types:         normalize(c.Types),
		Supertypes:    normalize(c.Supertypes),
		Subtypes:      normalize(c.Subtypes),
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		ManaCost:      c.ManaCost,
		ConvertedCost: int(c.ConvertedCost),
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

func CreateStringArray(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
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

	for _, c := range cards {
		// Not sure how to handle failure here
		_, err := tx.Exec("INSERT INTO cards (id, name, mana_cost, toughness, power, types, subtypes, supertypes, colors, cmc, rules, loyalty) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)", c.Id, c.Name, c.ManaCost, c.Toughness, c.Power, CreateStringArray(c.Types), CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes), CreateStringArray(c.Colors), c.ConvertedCost, c.Text, c.Loyalty)

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
