package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/gzip"
	_ "github.com/lib/pq"
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
	Id            string            `json:"id"`
	Href          string            `json:"url,omitempty"`
	Types         []string          `json:"types,omitempty"`
	Supertypes    []string          `json:"supertypes,omitempty"`
	Subtypes      []string          `json:"subtypes,omitempty"`
	Colors        []string          `json:"colors,omitempty"`
	FormatMap     map[string]string `json:"formats"`
	ConvertedCost int               `json:"cmc"`
	ManaCost      string            `json:"cost"`
	Text          string            `json:"text"`
	Power         string            `json:"power,omitempty"`
	Toughness     string            `json:"toughness,omitempty"`
	Loyalty       int               `json:"loyalty,omitempty"`
	Editions      []Edition         `json:"editions,omitempty"`
}

func (c *Card) Sets() []string {
	sets := []string{}
	for _, e := range c.Editions {
		sets = append(sets, e.SetId)
	}
	return ToUniqueLower(sets)
}

func (c *Card) Formats() []string {
	v := []string{}
	for format, _ := range c.FormatMap {
		v = append(v, format)
	}
	return ToUniqueLower(v)
}

func (c *Card) Status() []string {
	v := []string{}
	for _, status := range c.FormatMap {
		v = append(v, status)
	}
	return ToUniqueLower(v)
}

func (c *Card) Rarities() []string {
	r := []string{}
	for _, e := range c.Editions {
		r = append(r, e.Rarity)
	}
	return ToUniqueLower(r)
}

func (c *Card) MultiverseIds() []string {
	r := []string{}
	for _, e := range c.Editions {
		r = append(r, strconv.Itoa(e.MultiverseId))
	}
	return ToUniqueLower(r)
}


func (c *Card) Fill() {
	c.Href = fmt.Sprintf("%s/mtg/cards/%s", GetHostname(), c.Id)

    for i, _ := range c.Editions {
            c.Editions[i].Fill()
 }
}

type Edition struct {
	Set          string `json:"set" db:"set_name"`
	SetId        string `json:"-" db:"set_id"`
	CardId       string `json:"-" db:"card_id"`
	Watermark    string `json:"watermark,omitempty"`
	Rarity       string `json:"rarity"`
	Border       string `json:"-"`
	Artist       string `json:"artist"`
	MultiverseId int    `json:"multiverse_id" db:"eid"`
	Flavor       string `json:"flavor,omitempty"`
	Number       string `json:"number" db:"set_number"`
	Layout       string `json:"layout"`
	Href         string `json:"url,omitempty" db:"-"`
	ImageUrl     string `json:"image_url,omitempty" db:"-"`
	SetUrl       string `json:"set_url,omitempty" db:"-"`
}

func (e *Edition) Fill() {
	e.Href = fmt.Sprintf("%s/mtg/cards?multiverseid=%d", GetHostname(), e.MultiverseId)
	e.SetUrl = fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), e.SetId)
	e.ImageUrl = fmt.Sprintf("http://mtgimage.com/multiverseid/%d.jpg", e.MultiverseId)
}

type Set struct {
	Id     string `json:"id"`
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

// FIXME: What the fuck to do with this?
func specialFetch(db *sql.DB, cond Condition) ([]Card, error) {
	cards := []Card{}

	query := Select("record").From("cards").Where(cond).Limit(100).OrderBy("name", true)

	ql, items, err := query.ToSql()

	if err != nil {
		return cards, err
	}

	log.Println(ql, items)

	rows, err := db.Query(ql, items...)

	if err != nil {
		return cards, err
	}

	for rows.Next() {
		var blob []byte
		var card Card

		if err := rows.Scan(&blob); err != nil {
			return cards, err
		}

		err = json.Unmarshal(blob, &card)

		if err != nil {
			return cards, err
		}

        card.Fill()

		cards = append(cards, card)
	}

	if err := rows.Err(); err != nil {
		return cards, err
	}

	return cards, nil
}

func GetCards(db *sql.DB, req *http.Request, w http.ResponseWriter) (int, []byte) {
	cond, err, errors := ParseSearch(req.URL)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, Errors(errors...))
	}

	cards, err := specialFetch(db, cond)

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Cards not found"))
	}

	w.Header().Set("Link", LinkHeader(GetHostname(), req.URL, 0))

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

func GetTypes(db *sql.DB) (int, []byte) {
	types, err := FetchTerms(db, "types")

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

func GetCard(db *Database, params martini.Params) (int, []byte) {
	card, err := db.FetchCard(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Card not found"))
	}

	return JSON(http.StatusOK, card)
}

type Pong struct {
	Rally string `json:"rally"`
}

func Ping() (int, []byte) {
	return JSON(http.StatusOK, Pong{Rally: "serve"})
}

func GetEdition(db *Database, params martini.Params) (int, []byte) {
	cards, err := db.FetchEditions(params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Edition not found"))
	}

	return JSON(http.StatusOK, cards)
}

func NotFound() (int, []byte) {
	return JSON(http.StatusNotFound, Errors("No endpoint here"))
}

func Placeholder(params martini.Params) string {
	return "Hello world!"
}

func NewApi() *martini.Martini {
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
	r.Get("/mtg/editions/:id", GetEdition)
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
	return m
}

func main() {
	flag.Parse()

	db, err := sql.Open("postgres", "postgres://urza:power9@localhost/deckbrew?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

	if db.Ping() != nil {
		log.Fatal(db.Ping())
	}

	if flag.Arg(0) == "load" {
		err := FillDatabase(db, flag.Arg(1))

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Loaded all data into the database")
		return
	}

	m := NewApi()
	m.Map(db)
	m.Run()
}
