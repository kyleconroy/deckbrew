package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"log"
	"net/http"
	"os"
	"strings"
)

func GetHostname() string {
	hostname := os.Getenv("DECKBREW_HOSTNAME")

	if hostname == "" {
		return "api.deckbrew.com"
	}

	return hostname
}

// Import this eventually
type Card struct {
	Name             string    `json:"name" db:"name"`
	Id               string    `json:"id" db:"id"`
	JoinedTypes      string    `json:"-" db:"types"`
	JoinedSupertypes string    `json:"-" db:"supertypes"`
	JoinedSubtypes   string    `json:"-" db:"subtypes"`
	JoinedColors     string    `json:"-" db:"colors"`
	Types            []string  `json:"types,omitempty" db:"-"`
	Supertypes       []string  `json:"supertypes,omitempty" db:"-"`
	Subtypes         []string  `json:"subtypes,omitempty" db:"-"`
	Colors           []string  `json:"colors,omitempty" db:"-"`
	ConvertedCost    int       `json:"cmc" db:"cmc"`
	ManaCost         string    `json:"cost" db:"mana_cost"`
	Text             string    `json:"text" db:"rules"`
	Power            string    `json:"power,omitempty" db:"power"`
	Toughness        string    `json:"toughness,omitempty" db:"toughness"`
	Loyalty          int       `json:"loyalty,omitempty" db:"loyalty"`
	Href             string    `json:"url,omitempty"`
	Editions         []Edition `json:"editions,omitempty"`
}

func explode(types string) []string {
	if types == "" {
		return []string{}
	} else {
		return strings.Split(types, ",")
	}
}

func (c *Card) Fill() {
	c.Href = fmt.Sprintf("http://%s/mtg/cards/%s", GetHostname(), c.Id)
	c.Types = explode(c.JoinedTypes)
	c.Supertypes = explode(c.JoinedSupertypes)
	c.Subtypes = explode(c.JoinedSubtypes)
	c.Colors = explode(c.JoinedColors)
}

type Edition struct {
	Set          string `json:"set" db:"set_name"`
	SetId        string `json:"-" db:"set_id"`
	CardId       string `json:"-" db:"card_id"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseId int    `json:"multiverse_id" db:"id"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number" db:"set_number"`
	Layout       string `json:"layout"`
	Href         string `json:"url,omitempty" db:"-"`
	ImageUrl     string `json:"image_url,omitempty" db:"-"`
	SetUrl       string `json:"set_url,omitempty" db:"-"`
}

func (e *Edition) Fill() {
	e.Href = fmt.Sprintf("http://%s/mtg/editions/%d", GetHostname(), e.MultiverseId)
	e.SetUrl = fmt.Sprintf("http://%s/mtg/sets/%s", GetHostname(), e.SetId)
	e.ImageUrl = fmt.Sprintf("http://mtgimage.com/multiverseid/%d.jpg", e.MultiverseId)
}

type Set struct {
	Id     string `json:"id"`
	Name   string `json:"Name"`
	Border string `json:"Border"`
	Type   string `json:"type"`
	Href   string `json:"url" db:"-"`
}

func (s *Set) Fill() {
	s.Href = fmt.Sprintf("http://%s/mtg/sets/%s", GetHostname(), s.Id)
}

func JSON(code int, val interface{}) (int, []byte) {
	blob, err := json.Marshal(val)

	if err != nil {
		return 500, []byte("INTERNAL SERVER ERROR")
	}

	return code, blob
}

func GetCards(db *Database, req *http.Request) (int, []byte) {
	q, err := NewQuery(req)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, "")
	}

	cards, err := db.FetchCards(q)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, cards)
}

func GetSets(db *Database) (int, []byte) {
	sets, err := db.FetchSets()

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, sets)
}

func GetSet(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchSet(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, card)
}

func GetCard(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchCard(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, card)
}

func GetEdition(db *Database, params martini.Params) (int, []byte) {
	cards, err := db.FetchEditions(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, cards)
}

func Placeholder(params martini.Params) string {
	return "Hello world!"
}

func main() {
	flag.Parse()

	db, err := Open("postgres://urza:power9@localhost/deckbrew?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

	if flag.Arg(0) == "load" {
		collection, err := LoadCollection(flag.Arg(1))

		if err != nil {
			log.Fatal(err)
		}

		err = db.Load(collection)

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Loaded all data into the database")
		return
	}

	m := martini.New()

	// Setup middleware
	m.Use(martini.Recovery())
	m.Use(martini.Logger())
	m.Use(gzip.All())
	m.Use(func(c martini.Context, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public,max-age=3600")
		w.Header().Set("License", "Card names and text are all copyright Wizards of the Coast. This website is not affiliated with Wizards of the Coast in any way.")
	})

	r := martini.NewRouter()

	r.Get("/mtg/cards", GetCards)
	r.Get("/mtg/cards/:id", GetCard)
	r.Get("/mtg/editions/:id", GetEdition)
	r.Get("/mtg/sets", GetSets)
	r.Get("/mtg/sets/:id", GetSet)

	m.Action(r.Handle)
	m.Map(&db)
	m.Run()
}
