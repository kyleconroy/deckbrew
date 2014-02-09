package main

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
)

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
		CardId:       c.Id(),
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
		Id:            c.Id(),
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

func TransformCollection(collection MTGCollection) ([]Set, []Card) {
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

	return sets, cards
}

func AddFormat(c *Card, f *MTGFormat) {
	if c.FormatMap == nil {
		c.FormatMap = map[string]string{}
	}

	update := func(cd *Card, set, status string) {
		cd.FormatMap[set] = status
		cd.Formats = ToUniqueLower(append(c.Formats, set))
		cd.Status = ToUniqueLower(append(c.Status, status))
	}

	for _, edition := range c.Editions {
		if edition.SetId == "UNH" || edition.SetId == "UGL" {
			return
		}
	}

	for _, b := range f.Banned {
		if c.Id == b.Id {
			update(c, f.Name, "banned")
			return
		}
	}

	for _, r := range f.Restricted {
		if c.Id == r.Id {
			update(c, f.Name, "restricted")
			return
		}
	}

	if len(f.Sets) == 0 {
		update(c, f.Name, "legal")
		return
	}

	for _, edition := range c.Editions {
		for _, format_set := range f.Sets {
			if strings.ToUpper(format_set) == strings.ToUpper(edition.SetId) {
				update(c, f.Name, "legal")
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

	sets, cards := TransformCollection(collection)

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
			"id", "name", "record", "rules",
            "rarities", "types",
            "subtypes", "supertypes",
			"colors", "sets",
		}

		q := Insert(columns, "cards")

		_, err = tx.Exec(q, c.Id, c.Name, blob, c.Text,
                CreateStringArray(c.Rarities), CreateStringArray(c.Types),
                CreateStringArray(c.Subtypes), CreateStringArray(c.Supertypes),
                CreateStringArray(c.Colors), CreateStringArray(c.Sets))

		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
}

func LoadFormats(paths ...string) ([]MTGFormat, error) {
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

func FillDatabase(db *sql.DB, path string) error {
	collection, err := LoadCollection(path)

	if err != nil {
		return err
	}

	err = CreateCollection(db, collection)

	if err != nil {
		return err
	}

	return nil
}
