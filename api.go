package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func GetHostname() string {
	hostname := os.Getenv("DECKBREW_HOSTNAME")

	if hostname == "" {
		return "https://api.deckbrew.com"
	}

	return hostname
}

// Import this eventually
type Card struct {
	Name          string            `json:"name"`
	Id            string            `json:"id" bson:"_id"`
	Href          string            `json:"url" bson:"-"`
	Types         []string          `json:"types,omitempty"`
	Supertypes    []string          `json:"supertypes,omitempty"`
	Subtypes      []string          `json:"subtypes,omitempty"`
	Colors        []string          `json:"colors,omitempty"`
	Rarities      []string          `json:"-"`
	Sets          []string          `json:"-"`
	ConvertedCost int               `json:"cmc"`
	ManaCost      string            `json:"cost"`
	Text          string            `json:"text"`
	Power         string            `json:"power,omitempty"`
	Toughness     string            `json:"toughness,omitempty"`
	Loyalty       int               `json:"loyalty,omitempty"`
	Formats       map[string]string `json:"formats"`
	Editions      []Edition         `json:"editions,omitempty"`
}

func (c *Card) Fill() {
	c.Href = fmt.Sprintf("%s/mtg/cards/%s", GetHostname(), c.Id)

	for i, _ := range c.Editions {
		c.Editions[i].Fill()
	}
}

type Edition struct {
	Set          string `json:"set"`
	SetId        string `json:"-"`
	CardId       string `json:"-" bson:"-"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseId int    `json:"multiverse_id"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number"`
	Layout       string `json:"layout"`
	Href         string `json:"url,omitempty" bson:"-"`
	ImageUrl     string `json:"image_url,omitempty" bson:"-"`
	SetUrl       string `json:"set_url,omitempty" bson:"-"`
}

func (e *Edition) Fill() {
	e.Href = fmt.Sprintf("%s/mtg/cards?multiverseid=%d", GetHostname(), e.MultiverseId)
	e.SetUrl = fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), e.SetId)
	e.ImageUrl = fmt.Sprintf("http://mtgimage.com/multiverseid/%d.jpg", e.MultiverseId)
}

type Set struct {
	Id     string `json:"id" bson:"_id"`
	Name   string `json:"name"`
	Border string `json:"border"`
	Type   string `json:"type"`
	Href   string `json:"url" db:"-"`
}

func (s *Set) Fill() {
	s.Href = fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), s.Id)
}

func JSON(code int, val interface{}) (int, []byte) {
	blob, err := json.Marshal(val)

	if err != nil {
		return 500, []byte(`{"error": "Internal server error :("}"`)
	}

	return code, blob
}

func Errors(errors ...string) ApiError {
	return ApiError{Errors: errors}
}

func LinkHeader(host string, u *url.URL, page int) string {
	if page == 0 {
		qstring := u.Query()
		qstring.Set("page", "1")
		return fmt.Sprintf("<%s%s?%s>; rel=\"next\"", host, u.Path, qstring.Encode())
	} else {
		qstring := u.Query()

		qstring.Set("page", strconv.Itoa(page-1))
		prev := fmt.Sprintf("<%s%s?%s>; rel=\"prev\"", host, u.Path, qstring.Encode())

		qstring.Set("page", strconv.Itoa(page+1))
		next := fmt.Sprintf("<%s%s?%s>; rel=\"next\"", host, u.Path, qstring.Encode())

		return prev + ", " + next
	}
}

type ApiError struct {
	Errors []string `json:"errors"`
}

func GetCards(db *mgo.Database, req *http.Request, w http.ResponseWriter) (int, []byte) {
	q, err, errors := ParseSearch(req.URL)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, Errors(errors...))
	}

	page, err := CardsPaging(req.URL)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, Errors(err.Error()))
	}

	collection := db.C("cards")

	var cards []Card

	err = collection.Find(q).Limit(100).Skip(100 * page).Sort("name").All(&cards)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Card not found"))
	}

	if len(cards) == 0 {
		return JSON(http.StatusOK, []Card{})
	}

	for i, _ := range cards {
		cards[i].Fill()
	}

	w.Header().Set("Link", LinkHeader(GetHostname(), req.URL, page))

	return JSON(http.StatusOK, cards)
}
func GetSupertypes(db *Database) (int, []byte) {
	types, err := db.FetchTerms("supertypes")

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Supertypes not found"))
	}

	return JSON(http.StatusOK, types)
}

func GetColors(db *Database) (int, []byte) {
	types, err := db.FetchTerms("colors")

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Colors not found"))
	}

	return JSON(http.StatusOK, types)
}

func GetSubtypes(db *Database) (int, []byte) {
	types, err := db.FetchTerms("subtypes")

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Subtypes not found"))
	}

	return JSON(http.StatusOK, types)
}

func GetTypes(db *Database) (int, []byte) {
	types, err := db.FetchTerms("types")

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Types not found"))
	}

	return JSON(http.StatusOK, types)
}

func GetSets(db *Database) (int, []byte) {
	sets, err := db.FetchSets()

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Sets not found"))
	}

	return JSON(http.StatusOK, sets)
}

func GetSet(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchSet(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Set not found"))
	}

	return JSON(http.StatusOK, card)
}

func GetCard(db *mgo.Database, params martini.Params) (int, []byte) {
	collection := db.C("cards")

	var card Card

	err := collection.Find(bson.M{"_id": params["id"]}).One(&card)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Card not found"))
	}

	card.Fill()

	return JSON(http.StatusOK, card)
}

type Pong struct {
	Rally string `json:"rally"`
}

func Ping() (int, []byte) {
	return JSON(http.StatusOK, Pong{Rally: "serve"})
}

func NotFound() (int, []byte) {
	return JSON(http.StatusNotFound, Errors("No endpoint here"))
}

func Placeholder(params martini.Params) string {
	return "Hello world!"
}

func NewApi(db *Database) *martini.Martini {
	session, err := mgo.Dial("localhost:27017")

	if err != nil {
		log.Fatal(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	newdb := session.DB("deckbrew")

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
		w.Header().Set("Cache-Control", "public,max-age=3600")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("License", "The textual information presented through this API about Magic: The Gathering is copyrighted by Wizards of the Coast.")
		w.Header().Set("Disclaimer", "This API is not produced, endorsed, supported, or affiliated with Wizards of the Coast.")
	})

	r := martini.NewRouter()

	r.Get("/ping", Ping)
	r.Get("/mtg/cards", GetCards)
	r.Get("/mtg/cards/:id", GetCard)
	r.Get("/mtg/sets", GetSets)
	r.Get("/mtg/sets/:id", GetSet)
	r.Get("/mtg/colors", GetColors)
	r.Get("/mtg/supertypes", GetSupertypes)
	r.Get("/mtg/subtypes", GetSubtypes)
	r.Get("/mtg/types", GetTypes)
	r.NotFound(NotFound)

	//They can just download the mtgjson dump
	//r.Get("/mtg/editions", GetEditions)

	m.Action(r.Handle)
	m.Map(db)
	m.Map(newdb)

	return m
}

func main() {
	flag.Parse()

	db, err := Open("postgres://urza:power9@localhost/deckbrew?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

	if flag.Arg(0) == "load" {

		err = NewLoad(flag.Arg(1))

		if err != nil {
			log.Fatal(err)
		}

		return
	}

	m := NewApi(&db)
	m.Run()
}
