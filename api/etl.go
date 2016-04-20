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

	"github.com/kyleconroy/deckbrew/config"
	_ "github.com/lib/pq"
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
	switch r {
	case "mythic rare":
		return "mythic"
	case "basic land":
		return "basic"
	default:
		return r
	}
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
		FormatMap:     TransformLegalities(c.Legalities),
		ConvertedCost: int(c.ConvertedCost),
	}
}

func TransformCollection(collection MTGCollection) ([]Set, []Card) {
	cards := []Card{}
	ids := map[string]Card{}
	editions := []Edition{}
	sets := []Set{}

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

func existingSet(sets []Set, id string) bool {
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

func CreateCollection(db *cql.DB, collection MTGCollection) error {
	ctx := context.TODO()
	sets, cards := TransformCollection(collection)

	// Load the current cards and sets
	currentSets, err := FetchSets(ctx, db)
	if err != nil {
		return err
	}

	currentCards, err := FetchCardIDs(ctx, db)
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
		_, err := tx.Exec("INSERT INTO sets (id, name, border, type) VALUES ($1, $2, $3, $4)",
			s.Id, s.Name, s.Border, s.Type)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error intserting set %+v %s", s, err)
		}
	}

	// TODO: The code below won't handle reprints, as we won't update the card record
	// when it's been printed in a new set. Figure this out some how
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
		columns := []string{
			"id", "name", "record", "rules", "mana_cost", "cmc",
			"power", "toughness", "loyalty", "multicolor", "rarities",
			"types", "subtypes", "supertypes", "colors", "sets",
			"formats", "status", "mids",
		}
		if existingCard(currentCards, c.Id) {
			q := Update(columns[1:], "cards")
			_, err = tx.Exec(q, c.Name, blob, c.Text, c.ManaCost, c.ConvertedCost,
				c.Power, c.Toughness, c.Loyalty, c.Multicolor(),
				CreateStringArray(c.Rarities()), CreateStringArray(c.Types),
				CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes),
				CreateStringArray(c.Colors), CreateStringArray(c.Sets()),
				CreateStringArray(c.Formats()), CreateStringArray(c.Status()),
				CreateStringArray(c.MultiverseIds()), c.Id)
		} else {
			q := Insert(columns, "cards")
			_, err = tx.Exec(q, c.Id, c.Name, blob, c.Text, c.ManaCost, c.ConvertedCost,
				c.Power, c.Toughness, c.Loyalty, c.Multicolor(),
				CreateStringArray(c.Rarities()), CreateStringArray(c.Types),
				CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes),
				CreateStringArray(c.Colors), CreateStringArray(c.Sets()),
				CreateStringArray(c.Formats()), CreateStringArray(c.Status()),
				CreateStringArray(c.MultiverseIds()))
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

	return CreateCollection(cfg.DB, collection)
}
