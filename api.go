package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"log"
	"net/http"
	"strings"
)

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
	c.Href = "http://localhost:3000/cards/" + c.Id
	c.Types = explode(c.JoinedTypes)
	c.Supertypes = explode(c.JoinedSupertypes)
	c.Subtypes = explode(c.JoinedSubtypes)
	c.Colors = explode(c.JoinedColors)
}

type Edition struct {
	Set          string `json:"-" db:"magicset"`
	CardId       string `json:"-" db:"card_id"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseId int    `json:"multiverse_id" db:"id"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number" db:"magicnumber"`
	Layout       string `json:"layout"`
	Href         string `json:"url,omitempty" db:"-"`
	ImageUrl     string `json:"image_url,omitempty" db:"-"`
}

func (e *Edition) Fill() {
	e.Href = fmt.Sprintf("http://localhost:3000/editions/%d", e.MultiverseId)
	e.ImageUrl = fmt.Sprintf("http://mtgimage.com/multiverseid/%d.jpg", e.MultiverseId)
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

func GetCard(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchCard(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, "")
	}

	return JSON(http.StatusOK, card)
}

func GetCardEditions(db *Database, params martini.Params) string {
	return "Hello world!"
}

func GetEditions(db *Database) string {
	return "Hello world!"
}

func GetEdition(db *Database, params martini.Params) string {
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
	})

	r := martini.NewRouter()

	r.Get("/cards", GetCards)
	r.Get("/cards/:id", GetCard)
	r.Get("/cards/:id/editions", GetCardEditions)
	r.Get("/editions", GetEditions)
	r.Get("/editions/:id", GetEdition)

	m.Action(r.Handle)
	m.Map(&db)
	m.Run()
}
