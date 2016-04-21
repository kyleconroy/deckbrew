package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/net/context"

	"stackmachine.com/cql"

	"github.com/kyleconroy/deckbrew/brew"
	"github.com/kyleconroy/deckbrew/config"
	_ "github.com/lib/pq"
)

func sarray(values []string) string {
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

func transformRarity(rarity string) string {
	r := strings.ToLower(rarity)
	switch r {
	case "mythic rare":
		return "mythic"
	case "basic land":
		return "basic"
	default:
		return r
	}
}

func TransformEdition(s MTGSet, c MTGCard) brew.Edition {
	return brew.Edition{
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
func TransformSet(s MTGSet) brew.Set {
	return brew.Set{
		Name:   s.Name,
		Id:     s.Code,
		Border: s.Border,
		Type:   s.Type,
	}
}

func TransformCard(c MTGCard) brew.Card {
	return brew.Card{
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
		FormatMap:     TransformLegalities(c.Legalities),
		ConvertedCost: int(c.ConvertedCost),
	}
}

func TransformCollection(collection MTGCollection) ([]brew.Set, []brew.Card) {
	cards := []brew.Card{}
	ids := map[string]brew.Card{}
	editions := []brew.Edition{}
	sets := []brew.Set{}

	for _, set := range collection {
		if strings.HasPrefix(set.Name, "p") {
			continue
		}

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

	return sets, cards
}

func TransformLegalities(lgs MTGLegalities) map[string]string {
	formats := map[string]string{}
	if lgs.Standard != "" {
		formats["standard"] = strings.ToLower(lgs.Standard)
	}
	if lgs.Modern != "" {
		formats["modern"] = strings.ToLower(lgs.Modern)
	}
	if lgs.Legacy != "" {
		formats["legacy"] = strings.ToLower(lgs.Legacy)
	}
	if lgs.Vintage != "" {
		formats["vintage"] = strings.ToLower(lgs.Vintage)
	}
	if lgs.Commander != "" {
		formats["commander"] = strings.ToLower(lgs.Commander)
	}
	return formats
}

func existingSet(sets []brew.Set, id string) bool {
	for _, cs := range sets {
		if cs.Id == id {
			return true
		}
	}
	return false
}

func existingCard(ids []string, id string) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

func fetchCardIDs(ctx context.Context, db *cql.DB) ([]string, error) {
	ids := []string{}

	rows, err := db.QueryC(ctx, "SELECT id FROM cards")
	if err != nil {
		return ids, err
	}

	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

const queryInsertSet = `
INSERT INTO sets (id, name, border, type) VALUES ($1, $2, $3, $4)
`

const queryInsertCard = `
INSERT INTO cards (
  id, name, record, rules, mana_cost, cmc,
  power, toughness, loyalty, multicolor, rarities,
  types, subtypes, supertypes, colors, sets,
  formats, status, mids
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18, $19
)
`

const queryUpdateCard = `
UPDATE cards SET (
  name, record, rules, mana_cost, cmc,
  power, toughness, loyalty, multicolor, rarities,
  types, subtypes, supertypes, colors, sets,
  formats, status, mids
) = (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18
)
WHERE id = $19
`

func CreateCollection(db *cql.DB, r brew.Reader, collection MTGCollection) error {
	ctx := context.TODO()
	sets, cards := TransformCollection(collection)

	// Load the current cards and sets
	currentSets, err := r.GetSets(ctx)
	if err != nil {
		return err
	}

	currentCards, err := fetchCardIDs(ctx, db)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, s := range sets {
		if existingSet(currentSets, s.Id) {
			continue
		}
		_, err := tx.Exec(queryInsertSet, s.Id, s.Name, s.Border, s.Type)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error intserting set %+v %s", s, err)
		}
	}

	i := 0
	for _, c := range cards {
		if i >= 1000 {
			log.Println("Added 1000 cards to the database")
		}
		blob, err := json.Marshal(c)
		if err != nil {
			tx.Rollback()
			return err
		}
		if existingCard(currentCards, c.Id) {
			_, err = tx.Exec(queryUpdateCard,
				c.Name, blob, c.Text, c.ManaCost, c.ConvertedCost,
				c.Power, c.Toughness, c.Loyalty, c.Multicolor(),
				sarray(c.Rarities()), sarray(c.Types),
				sarray(c.Subtypes), sarray(c.Supertypes),
				sarray(c.Colors), sarray(c.Sets()),
				sarray(c.Formats()), sarray(c.Status()),
				sarray(c.MultiverseIds()), c.Id)
		} else {
			_, err = tx.Exec(queryInsertCard,
				c.Id, c.Name, blob, c.Text, c.ManaCost, c.ConvertedCost,
				c.Power, c.Toughness, c.Loyalty, c.Multicolor(),
				sarray(c.Rarities()), sarray(c.Types),
				sarray(c.Subtypes), sarray(c.Supertypes),
				sarray(c.Colors), sarray(c.Sets()),
				sarray(c.Formats()), sarray(c.Status()),
				sarray(c.MultiverseIds()))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting / updating card %+v %s", c, err)
		}
		i += 1
	}
	return tx.Commit()
}

// I probably should have just kept the Makefile
func DownloadCards(url, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	out, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(out, rc)
		if err != nil {
			return err
		}
		rc.Close()
	}
	return nil
}

func SyncCards() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	path := "cards.json"
	log.Println("downloading cards from mtgjson.com")
	err = DownloadCards("http://mtgjson.com/json/AllSets-x.json.zip", path)
	if err != nil {
		return err
	}
	log.Println("loading cards into database")
	collection, err := LoadCollection(path)
	if err != nil {
		return err
	}
	client, err := brew.NewReader(cfg)
	if err != nil {
		return err
	}

	return CreateCollection(cfg.DB, client, collection)
}
