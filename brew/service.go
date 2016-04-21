package brew

import (
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/context"
)

type Reader interface {
	GetCards(context.Context, Search, int) ([]Card, error)
	GetCardsByName(context.Context, string) ([]Card, error)
	GetCard(context.Context, string) (Card, error)
	GetRandomCardID(context.Context) (string, error)
	GetSets(context.Context) ([]Set, error)
	GetSet(context.Context, string) (Set, error)
	GetColors(context.Context) ([]string, error)
	GetSupertypes(context.Context) ([]string, error)
	GetSubtypes(context.Context) ([]string, error)
	GetTypes(context.Context) ([]string, error)
}

func toUniqueLower(things []string) []string {
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

type Search struct {
	Colors            []string
	Formats           []string
	IncludeMulticolor bool
	Multicolor        bool
	MultiverseIDs     []string
	Names             []string
	Rarities          []string
	Sets              []string
	Status            []string
	Subtypes          []string
	Supertypes        []string
	Rules             []string
	Types             []string
}

type Card struct {
	Name          string            `json:"name"`
	Id            string            `json:"id"`
	Href          string            `json:"url,omitempty"`
	StoreUrl      string            `json:"store_url"`
	Types         []string          `json:"types,omitempty"`
	Supertypes    []string          `json:"supertypes,omitempty"`
	Subtypes      []string          `json:"subtypes,omitempty"`
	Colors        []string          `json:"colors,omitempty"`
	ConvertedCost int               `json:"cmc"`
	ManaCost      string            `json:"cost"`
	Text          string            `json:"text"`
	Power         string            `json:"power,omitempty"`
	Toughness     string            `json:"toughness,omitempty"`
	Loyalty       int               `json:"loyalty,omitempty"`
	FormatMap     map[string]string `json:"formats"`
	Editions      []Edition         `json:"editions,omitempty"`
}

func (c *Card) Sets() []string {
	sets := []string{}
	for _, e := range c.Editions {
		sets = append(sets, e.SetId)
	}
	return toUniqueLower(sets)
}

func (c *Card) Formats() []string {
	v := []string{}
	for format, status := range c.FormatMap {
		if status == "legal" || status == "restricted" {
			v = append(v, format)
		}
	}
	return toUniqueLower(v)
}

func (c *Card) Status() []string {
	v := []string{}
	for _, status := range c.FormatMap {
		v = append(v, status)
	}
	return toUniqueLower(v)
}

func (c *Card) Rarities() []string {
	r := []string{}
	for _, e := range c.Editions {
		r = append(r, e.Rarity)
	}
	return toUniqueLower(r)
}

func (c *Card) MultiverseIds() []string {
	r := []string{}
	for _, e := range c.Editions {
		r = append(r, strconv.Itoa(e.MultiverseId))
	}
	return toUniqueLower(r)
}

func (c *Card) Multicolor() bool {
	return len(c.Colors) > 1
}

// Don't expose this
func (c *Card) Fill(r router) {
	c.Href = r.CardURL(c.Id)
	c.StoreUrl = TCGCardURL(c)

	for i, _ := range c.Editions {
		e := &c.Editions[i]
		e.Href = r.EditionURL(e.MultiverseId)
		e.SetUrl = r.SetURL(e.SetId)
		e.ImageUrl = r.EditionImageURL(e.MultiverseId)
		e.HTMLUrl = r.EditionHtmlURL(e.MultiverseId)
		e.StoreUrl = TCGEditionURL(c, e)
		e.Price = &Price{
			Low:     0,
			Average: 0,
			High:    0,
		}
	}
}

type Edition struct {
	Set          string `json:"set"`
	SetId        string `json:"set_id"`
	CardId       string `json:"-"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseId int    `json:"multiverse_id"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number"`
	Layout       string `json:"layout"`
	Price        *Price `json:"price,omitempty"`
	Href         string `json:"url,omitempty"`
	ImageUrl     string `json:"image_url,omitempty"`
	SetUrl       string `json:"set_url,omitempty"`
	StoreUrl     string `json:"store_url"`
	HTMLUrl      string `json:"html_url"`
}

type Price struct {
	Low     int `json:"low"`
	Average int `json:"median"`
	High    int `json:"high"`
}

type Set struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Border     string `json:"border"`
	Type       string `json:"type"`
	Href       string `json:"url"`
	CardsUrl   string `json:"cards_url"`
	PriceGuide string `json:"-"`
	Priced     bool   `json:"-"`
}

func (s *Set) Fill(r router) {
	s.Href = r.SetURL(s.Id)
	s.CardsUrl = r.SetCardsURL(s.Id)
}
