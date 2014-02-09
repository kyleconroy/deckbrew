package main

import (
	"crypto/md5"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/url"
	"regexp"
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
	Rarity     []string
	Sets       []string
	Name       string
	Formats    []string
	Status     []int
}

func (q *Query) AddWhere(expr Expression) Expression {
	c := []Condition{}

	over := func(column string, strs []string) Condition {
		sort.Strings(strs)
		return Overlap(column, CreateStringArray(strs))
	}

	c = append(c, over("types", q.Types))

	if len(q.Subtypes) != 0 {
		c = append(c, over("subtypes", q.Subtypes))
	}

	if len(q.Supertypes) != 0 {
		c = append(c, over("supertypes", q.Supertypes))
	}

	if q.Name != "" {
		c = append(c, Regexp("name", "~*", q.Name))
	}

	if len(q.Sets) != 0 {
		c = append(c, over("sets", q.Sets))
	}

	if len(q.Rarity) != 0 {
		c = append(c, over("rarities", q.Rarity))
	}

	if len(q.Colors) != 0 {
		c = append(c, over("colors", q.Colors))
	}

	if len(q.Formats) > 0 {
		or_conds := []Condition{}

		for _, format := range q.Formats {
			or_conds = append(or_conds, Gt(format, 0))
		}

		c = append(c, Or(or_conds...))
	}

	if len(q.Status) > 0 {
		formats := q.Formats

		if len(formats) == 0 {
			formats = []string{"commander", "vintage", "legacy", "standard", "modern"}
		}

		or_conds := []Condition{}

		for _, status := range q.Status {
			for _, format := range formats {
				or_conds = append(or_conds, Eq(format, status))
			}
		}

		c = append(c, Or(or_conds...))
	}

	return expr.Where(And(c...)).OrderBy("name", true).Limit(100).Offset(q.PageOffset())
}

func (q *Query) PageOffset() int {
	return q.Page * 100
}

func extractSubtypes(args url.Values) ([]string, error) {
	return args["subtype"], nil
}

func extractLegal(args url.Values, key string) (int, error) {
	allowed := map[string]int{
		"legal":  1,
		"banned": 3,
	}

	if key == "vintage" {
		allowed["restrcited"] = 2
	}

	item := args.Get(key)

	if item == "" {
		return 0, nil
	}

	if _, found := allowed[item]; !found {
		return 0, fmt.Errorf("The %s format doesn't supprt %s", key, item)
	}

	return allowed[item], nil
}

func extractInts(args url.Values, key string, allowed map[string]int) ([]int, error) {
	items := args[key]
	ints := []int{}

	if len(items) == 0 {
		return []int{}, nil
	}

	for _, t := range items {
		if allowed[t] == 0 {
			return ints, fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
		ints = append(ints, allowed[t])
	}

	return ints, nil
}

func extractItems(args url.Values, key string, allowed map[string]bool) ([]string, error) {
	items := args[key]

	if len(items) == 0 {
		return []string{}, nil
	}

	for _, t := range items {
		if !allowed[t] {
			return items, fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
	}

	return items, nil
}

func extractRarity(args url.Values) ([]string, error) {
	allowed := map[string]bool{
		"common":      true,
		"uncommon":    true,
		"rare":        true,
		"mythic rare": true,
		"special":     true,
		"basic land":  true,
	}
	return extractItems(args, "rarity", allowed)
}

func extractColors(args url.Values) ([]string, error) {
	allowed := map[string]bool{
		"red":   true,
		"blue":  true,
		"green": true,
		"black": true,
		"white": true,
	}
	return extractItems(args, "color", allowed)
}

func extractStatus(args url.Values) ([]int, error) {
	allowed := map[string]int{
		"legal":      1,
		"restricted": 2,
		"banned":     3,
	}
	return extractInts(args, "status", allowed)
}

func extractFormats(args url.Values) ([]string, error) {
	allowed := map[string]bool{
		"vintage":   true,
		"commander": true,
		"standard":  true,
		"legacy":    true,
		"modern":    true,
	}
	return extractItems(args, "format", allowed)
}

func extractSupertypes(args url.Values) ([]string, error) {
	allowed := map[string]bool{
		"legendary": true,
		"basic":     true,
		"world":     true,
		"snow":      true,
		"ongoing":   true,
	}
	return extractItems(args, "supertype", allowed)
}

func extractName(args url.Values) (string, error) {
	name := args.Get("name")

	if name == "" {
		return "", nil
	}

	if match, _ := regexp.MatchString("^[0-9A-Za-z,' ]+$", name); !match {
		return "", fmt.Errorf("The pattern %s can only contain letters, numbers, commas, single quotes and spaces")
	}

	return name, nil
}

func extractTypes(args url.Values) ([]string, error) {
	allowed := map[string]bool{
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

	items, err := extractItems(args, "type", allowed)

	if err != nil {
		return []string{}, err
	}

	if len(items) == 0 {
		return []string{
			"creature", "land", "enchantment", "sorcery",
			"instant", "planeswalker", "artifact",
		}, nil
	}

	return items, nil
}

type Search struct {
	Query map[string]interface{}
	Args  url.Values
}

func (s *Search) extractStrings(searchTerm, key string, allowed map[string]bool) error {
	items := s.Args[searchTerm]

	if len(items) == 0 {
		return nil
	}

	for _, t := range items {
		if !allowed[t] {
			return fmt.Errorf("The %s '%s' is not recognized", key, t)
		}
	}

	if len(items) == 1 {
		s.Query[key] = items[0]
	} else {
		s.Query[key] = map[string][]string{"$in": items}
	}

	return nil
}

func (s *Search) ParseMultiverseId() error {
	mid := s.Args.Get("multiverseid")

	if mid == "" {
		return nil
	}

	id, err := strconv.Atoi(mid)

	if err == nil {
		s.Query["editions.multiverseid"] = id
	}
	return err
}

func (s *Search) ParseSupertypes() error {
	return s.extractStrings("supertype", "supertypes", map[string]bool{
		"legendary": true,
		"basic":     true,
		"world":     true,
		"snow":      true,
		"ongoing":   true,
	})
}

func (s *Search) ParseSubtypes() error {
	sts := s.Args["subtype"]

	if len(sts) > 0 {
		s.Query["subtypes"] = map[string][]string{"$in": sts}
	}
	return nil
}

func (s *Search) ParseRarity() error {
	return s.extractStrings("rarity", "editions.rarity", map[string]bool{
		"common":      true,
		"uncommon":    true,
		"rare":        true,
		"mythic rare": true,
		"special":     true,
		"basic land":  true,
	})
}

func (s *Search) ParseTypes() error {
	err := s.extractStrings("type", "types", map[string]bool{
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
	})

	if err != nil {
		return err
	}

	if _, set := s.Query["types"]; !set {
		s.Query["types"] = map[string][]string{
			"$in": []string{"creature", "land", "enchantment",
				"sorcery", "instant", "planeswalker", "artifact"},
		}
	}

	return nil
}

func ParseSearch(u *url.URL) (interface{}, error, []string) {
	search := Search{Args: u.Query(), Query: map[string]interface{}{}}

	errs := []error{
		search.ParseMultiverseId(),
		search.ParseRarity(),
		search.ParseTypes(),
		search.ParseSupertypes(),
		search.ParseSubtypes(),
	}

	var err error
	results := []string{}

	for _, e := range errs {
		if e != nil {
			results = append(results, e.Error())
			err = fmt.Errorf("Errors while processing the search")
		}
	}

	return search.Query, err, results
}

func CardsPaging(u *url.URL) (int, error) {
	pagenum := u.Query().Get("page")

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

	q.Name, err = extractName(args)

	if err != nil {
		return q, err
	}

	q.Formats, err = extractFormats(args)

	if err != nil {
		return q, err
	}

	q.Status, err = extractStatus(args)

	if err != nil {
		return q, err
	}

	q.Sets = args["set"]
	q.Rarity, err = extractRarity(args)

	if err != nil {
		return q, err
	}

	return q, nil
}

type Database struct {
	conn *sqlx.DB
}

func (db *Database) ScanCard(c *Card, id string) error {
	return db.conn.Get(c, "SELECT name, cid, array_to_string(types, ',') AS types, array_to_string(subtypes, ',') AS subtypes, array_to_string(supertypes, ',') AS supertypes, array_to_string(colors, ',') AS colors, mana_cost, cmc, loyalty, rules, standard, modern, commander, vintage, legacy FROM cards WHERE cid=$1", id)
}

func (db *Database) FetchEditions(id string) ([]Card, error) {
	editions := []Edition{}
	cards := []Card{}

	err := db.conn.Select(&editions, "SELECT * FROM editions WHERE eid=$1 ORDER BY eid ASC", id)

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

type StringRow struct {
	T string
}

// SQL Injection possibility! Never call this function with
// user defined input
func (db *Database) FetchTerms(term string) ([]string, error) {
	types := []StringRow{}
	result := []string{}

	err := db.conn.Select(&types, "select distinct unnest("+term+") as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC")

	if err != nil {
		return result, err
	}

	for _, row := range types {
		result = append(result, row.T)
	}

	return result, nil
}

func (db *Database) FetchCards(q Query) ([]Card, error) {
	cards := []Card{}

	query := Select("name", "cid", "mana_cost", "cmc", "loyalty", "rules",
		"standard", "modern", "legacy", "vintage", "commander",
		"array_to_string(subtypes, ',') AS subtypes",
		"array_to_string(supertypes, ',') AS supertypes",
		"array_to_string(colors, ',') AS colors")
	query = q.AddWhere(query.From("cards"))

	sql, items, err := query.ToSql()

	if err != nil {
		return cards, err
	}

	err = db.conn.Select(&cards, sql, items...)

	if err != nil {
		return cards, err
	}

	for i, _ := range cards {
		cards[i].Fill()

		err = db.conn.Select(&cards[i].Editions, "SELECT * FROM editions WHERE card_id=$1 ORDER BY eid ASC", cards[i].Id)

		if err != nil {
			log.Println(err)
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

	err = db.conn.Select(&card.Editions, "SELECT * FROM editions WHERE card_id=$1 ORDER BY eid ASC", card.Id)

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

func UniqueToLower(things []string) []string {
	seen := map[string]bool{}
	sorted := []string{}

	for _, thing := range things {
		if _, found := seen[thing]; !found {
			sorted = append(sorted, strings.ToLower(thing))
			seen[thing] = true
		}
	}

	sort.Strings(sorted)
	return sorted
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

	// Denormalize
	c_rarity := map[string][]string{}
	c_sets := map[string][]string{}

	for _, set := range collection {
		sets = append(sets, TransformSet(set))

		for _, card := range set.Cards {
			newcard := TransformCard(card)
			newedition := TransformEdition(set, card)

			if _, found := ids[newcard.Id]; !found {
				ids[newcard.Id] = newcard
				cards = append(cards, newcard)
			}

			c_sets[newcard.Id] = append(c_sets[newcard.Id], newedition.SetId)
			c_rarity[newcard.Id] = append(c_rarity[newcard.Id], newedition.Rarity)

			editions = append(editions, newedition)
		}
	}

	for i, c := range cards {
		cards[i].Sets = UniqueToLower(c_sets[c.Id])
		cards[i].Rarities = UniqueToLower(c_rarity[c.Id])
	}

	for i, c := range cards {
		for _, edition := range editions {
			if edition.CardId == c.Id {
				cards[i].Editions = append(cards[i].Editions, edition)
			}
		}
	}

	return sets, cards, editions
}

func CreateStringArray(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
}

func (f *Format) CardStatus(c *Card) int {
	for _, card_set := range c.Sets {
		if card_set == "unh" || card_set == "ugl" {
			return 0
		}
	}

	for _, b := range f.Banned {
		if c.Id == b.Id {
			return 3
		}
	}

	for _, r := range f.Restricted {
		if c.Id == r.Id {
			return 2
		}
	}

	if len(f.Sets) == 0 {
		return 1
	}

	for _, card_set := range c.Sets {
		for _, format_set := range f.Sets {
			if format_set == card_set {
				return 1
			}
		}
	}

	return 0
}

// Given an array of cards, load them into the database
func (db *Database) Load(collection MTGCollection) error {
	tx := db.conn.MustBegin()

	sets, cards, editions := TransformCollection(collection)

	modern, err := LoadFormat("formats/modern.json")

	if err != nil {
		return err
	}

	standard, err := LoadFormat("formats/standard.json")

	if err != nil {
		return err
	}

	vintage, err := LoadFormat("formats/vintage.json")

	if err != nil {
		return err
	}

	legacy, err := LoadFormat("formats/legacy.json")

	if err != nil {
		return err
	}

	commander, err := LoadFormat("formats/commander.json")

	if err != nil {
		return err
	}

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
		_, err := tx.Exec("INSERT INTO cards (cid, name, mana_cost, toughness, power, types, subtypes, supertypes, colors, cmc, rules, loyalty, rarities, sets, modern, standard, vintage, legacy, commander) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)", c.Id, c.Name, c.ManaCost, c.Toughness, c.Power, CreateStringArray(c.Types), CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes), CreateStringArray(c.Colors), c.ConvertedCost, c.Text, c.Loyalty, CreateStringArray(c.Rarities), CreateStringArray(c.Sets), modern.CardStatus(&c), standard.CardStatus(&c), vintage.CardStatus(&c), legacy.CardStatus(&c), commander.CardStatus(&c))

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	for _, edition := range editions {
		// Not sure how to handle failure here
		_, err := tx.NamedExec("INSERT INTO editions (eid, card_id, set_name, watermark, rarity, border, artist, flavor, set_number, layout, set_id) VALUES (:eid, :card_id, :set_name, :watermark, :rarity, :border, :artist, :flavor, :set_number, :layout, :set_id)", &edition)

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}
