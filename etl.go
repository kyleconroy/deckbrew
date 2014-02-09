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

// FIXME: Add TX support
// FIXME: Add Sets
func CreateCollection(session *mgo.Session, collection MTGCollection) error {

	_, cards, _ := TransformCollection(collection)

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
		mgo.Index{Key: []string{"types"}},
		mgo.Index{Key: []string{"subtypes"}},
		mgo.Index{Key: []string{"supertypes"}},
		mgo.Index{Key: []string{"colors"}},
		mgo.Index{Key: []string{"cmc"}},
	}

	for _, index := range indexes {
		err := cardCollection.EnsureIndex(index)

		if err != nil {
			return err
		}
	}
	return nil
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
