package main

import (
	"encoding/json"
	"flag"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"log"
	"net/http"
)

// Import this eventually
type Card struct {
	Name string `json:"name" db:"name"`
	Id   string `json:"id" db:"id"`
	//Types         []string  `json:"types,omitempty" db:"types"`
	//Supertypes    []string  `json:"supertypes,omitempty" db:"supertypes"`
	//Subtypes      []string  `json:"subtypes,omitempty" db:"subtypes"`
	//ConvertedCost int8       `json:"cmc" db:"cmc"`
	//ManaCost      string    `json:"cost" db:"mana_cost"`
	//Text          string    `json:"text" db:"rules"`
	//Colors        []string  `json:"colors,omitempty" db:"colors"`
	//Power         string    `json:"power,omitempty" db:"power"`
	//Toughness     string    `json:"toughness,omitempty" db:"toughness"`
	//Loyalty       int8      `json:"loyalty,omitempty" db:"loyalty"`
	//Editions      []Edition `json:"editions,omitempty"`
}

type Edition struct {
	Set          string   `json:"set"`
	Watermark    string   `json:"watermark,omitempty"`
	Rarity       string   `json:"rarity"`
	Artist       string   `json:"artist"`
	MultiverseId int      `json:"multiverse_id"`
	Flavor       []string `json:"flavor,omitempty"`
	Number       string   `json:"number"`
}

func JSON(code int, val interface{}) (int, []byte) {
	blob, err := json.Marshal(val)

	if err != nil {
		return 500, []byte("INTERNAL SERVER ERROR")
	}

	return code, blob
}

func GetCards(db *Database, req *http.Request) (int, []byte) {
	cards, err := db.FetchCards(NewQuery(req))

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
