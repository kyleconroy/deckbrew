package main

import (
	"labix.org/v2/mgo"
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

	for i, c := range cards {
		for _, edition := range editions {
			if edition.CardId == c.Id {
				cards[i].Editions = append(cards[i].Editions, edition)
			}
		}
	}

	return sets, cards, editions
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

// FIXME: Add TX support
// FIXME: Add Sets
func CreateCollection(session *mgo.Session, collection MTGCollection) error {

	_, cards, _ := TransformCollection(collection)

	formats, err := LoadFormats("formats/commander.json", "formats/vintage.json",
		"formats/legacy.json", "formats/standard.json", "formats/modern.json")

	if err != nil {
		return err
	}

	for i, _ := range cards {
		for _, format := range formats {
			AddFormat(&cards[i], &format)
		}
	}

	cardCollection := session.DB("deckbrew").C("cards")

	for _, c := range cards {
		err := cardCollection.Insert(&c)

		if err != nil {
			return err
		}
	}

	return nil
}

func CreateIndexes(session *mgo.Session) error {
	cardCollection := session.DB("deckbrew").C("cards")

	indexes := []mgo.Index{
		mgo.Index{Key: []string{"name"}, Unique: true, DropDups: true},
		mgo.Index{Key: []string{"editions.multiverseid"}},
		mgo.Index{Key: []string{"editions.rarity"}},
		mgo.Index{Key: []string{"editions.setid"}},
		mgo.Index{Key: []string{"types"}},
		mgo.Index{Key: []string{"subtypes"}},
		mgo.Index{Key: []string{"supertypes"}},
		mgo.Index{Key: []string{"colors"}},
		mgo.Index{Key: []string{"formats"}},
		mgo.Index{Key: []string{"status"}},
		mgo.Index{Key: []string{"cmc"}},
		mgo.Index{Key: []string{"text"}},
	}

	for _, index := range indexes {
		err := cardCollection.EnsureIndex(index)

		if err != nil {
			return err
		}
	}
	return nil
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

func RecreateDatabase(session *mgo.Session, path string) error {
	err := session.DB("deckbrew").DropDatabase()

	if err != nil {
		return err
	}

	collection, err := LoadCollection(path)

	if err != nil {
		return err
	}

	err = CreateCollection(session, collection)

	if err != nil {
		return err
	}

	return CreateIndexes(session)
}
