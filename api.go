package main

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"io/ioutil"
	"log"
	"net/http"
)

// Import this eventually
type Card struct {
	Name           string    `json:"name"`
	Id             string    `json:"id"`
	Types          []string  `json:"types"`
	Subtypes       []string  `json:"subtypes,omitempty"`
	ConvertedCost  int       `json:"converted_cost"`
	ManaCost       string    `json:"mana_cost"`
	Special        string    `json:"special,omitempty"` //'flip', 'double-faced', 'split'
	PartnerCard    string    `json:"partner_card,omitempty"`
	RulesText      []string  `json:"rules_text"`
	ColorIndicator []string  `json:"color_indicator,omitempty"`
	Power          string    `json:"power,omitempty"`
	Toughness      string    `json:"toughness,omitempty"`
	Loyalty        int       `json:"loyalty,omitempty"`
	Editions       []Edition `json:"editions"`
}

func (c Card) Match(query Query) bool {
	for _, t := range c.Types {
		if query.Types[t] {
			return true
		}
	}
	return false
}

type Edition struct {
	Set          string   `json:"set"`
	Watermark    string   `json:"watermark,omitempty"`
	Rarity       string   `json:"rarity"`
	Artist       string   `json:"artist"`
	MultiverseId int      `json:"multiverse_id"`
	FlavorText   []string `json:"flavor_text,omitempty"`
	Number       string   `json:"number,omitempty"`
}

func JSON(code int, val interface{}) (int, []byte) {
	blob, err := json.Marshal(val)

	if err != nil {
		return 500, []byte("INTERNAL SERVER ERROR")
	}

	return code, blob
}

func GetCards(db *Database, req *http.Request) (int, []byte) {
	return JSON(http.StatusOK, db.FetchCards(NewQuery(req)))
}

func GetCard(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchCard(params["id"])

	if err != nil {
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
	blob, err := ioutil.ReadFile("cards.json")

	if err != nil {
		log.Fatal(err)
	}

	var db Database

	err = json.Unmarshal(blob, &db)

	if err != nil {
		log.Fatal(err)
	}

	m := martini.New()

	// Setup middleware
	m.Use(martini.Recovery())
	m.Use(martini.Logger())
	m.Use(gzip.All())
	m.Use(func(c martini.Context, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
