package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
)

func CreateStringArray(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
}

func ToSortedLower(things []string) []string {
	sorted := []string{}
	for _, thing := range things {
		sorted = append(sorted, strings.ToLower(strings.Replace(thing, ",", "", -1)))
	}
	sort.Strings(sorted)
	return sorted
}

func ToUniqueLower(things []string) []string {
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

func transformRarity(rarity string) string {
	r := strings.ToLower(rarity)

	if r == "mythic rare" {
		return "mythic"
	}

	if r == "basic land" {
		return "basic"
	}

	return r
}

func TransformEdition(s MTGSet, c MTGCard) Edition {
	return Edition{
		Set:          s.Name,
		SetId:        s.Code,
		Flavor:       c.Flavor,
		MultiverseId: c.MultiverseId,
		Watermark:    c.Watermark,
		Rarity:       transformRarity(c.Rarity),
		Artist:       c.Artist,
		Border:       c.Border,
		Layout:       c.Layout,
		Number:       c.Number,
		CardId:       Slug(c.Name),
	}
}

// FIXME: Add released dates
func TransformSet(s MTGSet) Set {
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
		Id:            Slug(c.Name),
		Text:          c.Text,
		Colors:        ToSortedLower(c.Colors),
		Types:         ToSortedLower(c.Types),
		Supertypes:    ToSortedLower(c.Supertypes),
		Subtypes:      ToSortedLower(c.Subtypes),
		Power:         c.Power,
		Toughness:     c.Toughness,
		Loyalty:       c.Loyalty,
		ManaCost:      c.ManaCost,
		ConvertedCost: int(c.ConvertedCost),
	}
}

func TransformCollection(collection MTGCollection, formats []MTGFormat) ([]Set, []Card) {
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

	for i, c := range cards {
		for _, edition := range editions {
			if edition.CardId == c.Id {
				cards[i].Editions = append(cards[i].Editions, edition)
			}
		}
	}

	for _, format := range formats {
		for i, _ := range cards {
			AddFormat(&cards[i], &format)
		}
	}

	return sets, cards
}

func AddFormat(c *Card, f *MTGFormat) {
	if c.FormatMap == nil {
		c.FormatMap = map[string]string{}
	}

	for _, special := range []string{"phenomenon", "plane", "scheme", "vanguard"} {
		for _, t := range c.Types {
			if t == special {
				return
			}
		}
	}

	for _, edition := range c.Editions {
		if edition.SetId == "UNH" || edition.SetId == "UGL" {
			return
		}
	}

	for _, b := range f.Banned {
		if c.Id == b.Id {
			c.FormatMap[f.Name] = "banned"
			return
		}
	}

	for _, r := range f.Restricted {
		if c.Id == r.Id {
			c.FormatMap[f.Name] = "restricted"
			return
		}
	}

	if len(f.Sets) == 0 {
		c.FormatMap[f.Name] = "legal"
		return
	}

	for _, edition := range c.Editions {
		for _, format_set := range f.Sets {
			if strings.ToUpper(format_set) == strings.ToUpper(edition.SetId) {
				c.FormatMap[f.Name] = "legal"
				return
			}
		}
	}
}

// FIXME: Add Sets
func CreateCollection(db *sql.DB, collection MTGCollection) error {
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	formats, err := LoadFormats()

	if err != nil {
		return err
	}

	sets, cards := TransformCollection(collection, formats)

	for _, s := range sets {
		_, err := tx.Exec("INSERT INTO sets (id, name, border, type) VALUES ($1, $2, $3, $4)",
			s.Id, s.Name, s.Border, s.Type)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, c := range cards {

		blob, err := json.Marshal(c)

		if err != nil {
			tx.Rollback()
			return err
		}

		columns := []string{
			"id", "name", "record", "rules", "mana_cost", "cmc",
			"power", "toughness", "loyalty", "multicolor", "rarities",
			"types", "subtypes", "supertypes", "colors", "sets",
			"formats", "status", "mids",
		}

		q := Insert(columns, "cards")

		_, err = tx.Exec(q, c.Id, c.Name, blob, c.Text, c.ManaCost, c.ConvertedCost,
			c.Power, c.Toughness, c.Loyalty, c.Multicolor(),
			CreateStringArray(c.Rarities()), CreateStringArray(c.Types),
			CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes),
			CreateStringArray(c.Colors), CreateStringArray(c.Sets()),
			CreateStringArray(c.Formats()), CreateStringArray(c.Status()),
			CreateStringArray(c.MultiverseIds()))

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func LoadFormats() ([]MTGFormat, error) {
	paths := []string{
		"formats/vintage.json", "formats/legacy.json",
		"formats/commander.json", "formats/standard.json",
		"formats/modern.json",
	}

	formats := []MTGFormat{}

	for _, path := range paths {
		f, err := LoadFormat(path)

		if err != nil {
			return formats, err
		}

		formats = append(formats, f)
	}

	return formats, nil
}

func exec(db *sql.DB, query string, args ...interface{}) {
	if _, err := db.Exec(query, args...); err != nil {
		log.Fatal(query, " ", err)
	}
}

func CreateDatabase(db *sql.DB) {
	user := os.Getenv("DATABASE_USER")
	pass := os.Getenv("DATABASE_PASSWORD")

	if user == "" || pass == "" {
		log.Fatal("DATABASE_USER and DATABASE_PASSWORD must be set")
	}

	exec(db, "DROP DATABASE IF EXISTS deckbrew")
	exec(db, "DROP USER IF EXISTS "+user)
	exec(db, "CREATE DATABASE deckbrew WITH template=template0 encoding='UTF8'")
	exec(db, "CREATE USER "+user+" WITH PASSWORD '"+pass+"'")
	exec(db, "GRANT ALL PRIVILEGES ON DATABASE deckbrew TO "+user)
}

func CreateTables(db *sql.DB) {
	user := os.Getenv("DATABASE_USER")

	exec(db, "CREATE EXTENSION pg_trgm")

	exec(db, `CREATE TABLE cards (
        id                varchar(150)   primary key,
        name              varchar(200)   DEFAULT '',
        mana_cost         varchar(45)    DEFAULT '',
        toughness         varchar(6)     DEFAULT '',
        power             varchar(6)     DEFAULT '',
        types             varchar(20)[]  DEFAULT '{}',
        rarities          varchar(15)[]  DEFAULT '{}',
        sets              varchar(3)[]   DEFAULT '{}',
        subtypes          varchar(100)[] DEFAULT '{}',
        supertypes        varchar(100)[] DEFAULT '{}',
        colors            varchar(5)[]   DEFAULT '{}',
        formats           varchar(9)[]   DEFAULT '{}',
        status            varchar(10)[]  DEFAULT '{}',
        mids              varchar(20)[]  DEFAULT '{}',
        multicolor        boolean        DEFAULT false,
        record            text           DEFAULT '',
        rules             text           DEFAULT '',
        loyalty           integer        DEFAULT 0,
        cmc               integer        DEFAULT 0)`)

	exec(db, `CREATE TABLE prices (
        id                integer        primary key,
        card_id           varchar(150)   DEFAULT '',
        name              varchar(200)   DEFAULT '',
        rarity            varchar(15)    DEFAULT '',
        set               varchar(3)     DEFAULT '',
        foil              boolean        DEFAULT false,
        price_low         integer        DEFAULT 0,
        price_median      integer        DEFAULT 0,
        price_high        integer        DEFAULT 0)`)

	exec(db, `CREATE TABLE sets (
        id                varchar(3) primary key,
        name              varchar(200) DEFAULT '',
        border            varchar(40) DEFAULT '',
        type              varchar(32) DEFAULT '')`)

	exec(db, "CREATE INDEX cards_name_id ON cards(id)")
	exec(db, "CREATE INDEX cards_power_index ON cards(power)")
	exec(db, "CREATE INDEX cards_toughness_index ON cards(toughness)")
	exec(db, "CREATE INDEX cards_names_sort_index ON cards(name)")
	exec(db, "CREATE INDEX cards_multicolor_index ON cards(multicolor)")
	exec(db, "CREATE INDEX cards_types_index ON cards USING GIN(types)")
	exec(db, "CREATE INDEX cards_subtypes_index ON cards USING GIN(subtypes)")
	exec(db, "CREATE INDEX cards_supertypes_index ON cards USING GIN(supertypes)")
	exec(db, "CREATE INDEX cards_colors_index ON cards USING GIN(colors)")
	exec(db, "CREATE INDEX cards_sets_index ON cards USING GIN(sets)")
	exec(db, "CREATE INDEX cards_rarities_index ON cards USING GIN(rarities)")
	exec(db, "CREATE INDEX cards_status_index ON cards USING GIN(status)")
	exec(db, "CREATE INDEX cards_formats_index ON cards USING GIN(formats)")
	exec(db, "CREATE INDEX cards_mids_index ON cards USING GIN(mids)")
	exec(db, "CREATE INDEX cards_names_index ON cards USING GIN(name gin_trgm_ops)")
	exec(db, "CREATE INDEX cards_rules_index ON cards USING GIN(rules gin_trgm_ops)")

	exec(db, "CREATE INDEX prices_id_index ON prices(id)")
	exec(db, "CREATE INDEX prices_card_index ON prices(card_id)")
	exec(db, "CREATE INDEX prices_set_index ON prices(set)")
	exec(db, "CREATE INDEX prices_rarity_index ON prices(rarity)")
	exec(db, "CREATE INDEX prices_low_index ON prices(low)")
	exec(db, "CREATE INDEX prices_median_index ON prices(median)")
	exec(db, "CREATE INDEX prices_high_index ON prices(high)")
	exec(db, "CREATE INDEX prices_names_trgm ON prices USING GIN(name gin_trgm_ops)")

	exec(db, "CREATE INDEX sets_type_index ON sets(type)")
	exec(db, "CREATE INDEX sets_border_index ON sets(border)")

	exec(db, "GRANT ALL PRIVILEGES ON TABLE cards TO "+user)
	exec(db, "GRANT ALL PRIVILEGES ON TABLE sets TO "+user)
}

func SyncDatabase(path string) error {
	host := os.Getenv("DATABASE_HOST")

	if host == "" {
		host = "localhost"
	}

	master, err := sql.Open("postgres", "host="+host+" sslmode=disable")

	if err != nil {
		return err
	}

	if master.Ping() != nil {
		log.Println("Can't create database")
		return master.Ping()
	}

	CreateDatabase(master)

	sdb, err := sql.Open("postgres", "host="+host+" dbname=deckbrew sslmode=disable")

	if err != nil {
		return err
	}

	if sdb.Ping() != nil {
		log.Println("Can't create tables")
		return sdb.Ping()
	}

	CreateTables(sdb)

	collection, err := LoadCollection(path)

	if err != nil {
		return err
	}

	db, err := GetDatabase()

	if err != nil {
		return err
	}

	err = CreateCollection(db, collection)

	if err != nil {
		return err
	}

	return nil
}

func DumpDatabase(inpath string, outpath string) error {
	collection, err := LoadCollection(inpath)

	if err != nil {
		return err
	}

	formats, err := LoadFormats()

	if err != nil {
		return err
	}

	_, cards := TransformCollection(collection, formats)

	if err != nil {
		return err
	}

	blob, err := json.Marshal(cards)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(outpath, blob, 0644)
}
