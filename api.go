package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/gzip"
)

func GetHostname() string {
	hostname := os.Getenv("DECKBREW_HOSTNAME")
	if hostname == "" {
		return "http://localhost:3000"
	}
	return hostname
}

func ReverseCard(id string) string {
	return fmt.Sprintf("%s/mtg/cards/%s", GetHostname(), id)
}

func ReverseEdition(id int) string {
	return fmt.Sprintf("%s/mtg/cards?multiverseid=%d", GetHostname(), id)
}

func ReverseSet(id string) string {
	return fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), id)
}

func MTGImageURL(id int) string {
	return fmt.Sprintf("https://image.deckbrew.com/mtg/multiverseid/%d.jpg", id)
}

func Slug(name string) string {
	re := regexp.MustCompile(`[,.'"?:()]`)
	d := strings.Replace(strings.ToLower(name), " ", "-", -1)
	return re.ReplaceAllLiteralString(d, "")
}

// Import this eventually
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
	return ToUniqueLower(sets)
}

func (c *Card) Formats() []string {
	v := []string{}
	for format, status := range c.FormatMap {
		if status == "legal" || status == "restricted" {
			v = append(v, format)
		}
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

func (c *Card) Multicolor() bool {
	return len(c.Colors) > 1
}

func (c *Card) Fill(pl *PriceList) {
	c.Href = ReverseCard(c.Id)
	c.StoreUrl = TCGCardURL(c)

	for i, _ := range c.Editions {
		e := &c.Editions[i]
		e.Href = ReverseEdition(e.MultiverseId)
		e.SetUrl = ReverseSet(e.SetId)
		e.ImageUrl = MTGImageURL(e.MultiverseId)
		e.StoreUrl = TCGEditionURL(c, e)
		e.Price = pl.GetPrice(e.MultiverseId)
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
}

type Price struct {
	Low     int `json:"low"`
	Average int `json:"median"`
	High    int `json:"high"`
}

type Set struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Border   string `json:"border"`
	Type     string `json:"type"`
	Href     string `json:"url"`
	CardsUrl string `json:"cards_url"`
}

func (s *Set) Fill() {
	s.Href = fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), s.Id)
	s.CardsUrl = fmt.Sprintf("%s/mtg/cards?set=%s", GetHostname(), s.Id)
}

func JSON(code int, val interface{}) (int, []byte) {
	blob, err := json.MarshalIndent(val, "", "  ")

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

func HandleCards(db *sql.DB, pl *PriceList, req *http.Request, w http.ResponseWriter) (int, []byte) {
	cond, err, errors := ParseSearch(req.URL)
	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, Errors(errors...))
	}
	page, err := CardsPaging(req.URL)
	if err != nil {
		log.Println(err)
		return JSON(http.StatusBadRequest, Errors(errors...))
	}
	cards, err := FetchCards(db, cond, page)
	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Cards not found"))
	}
	for i, _ := range cards {
		cards[i].Fill(pl)
	}
	w.Header().Set("Link", LinkHeader(GetHostname(), req.URL, page))
	return JSON(http.StatusOK, cards)
}

func HandleRandomCard(db *sql.DB, w http.ResponseWriter, r *http.Request) (int, []byte) {
	var card string
	err := db.QueryRow("SELECT id FROM cards ORDER BY RANDOM() LIMIT 1").Scan(&card)
	switch {
	case err == sql.ErrNoRows:
		return JSON(http.StatusNotFound, Errors("No random card can be found"))
	case err != nil:
		log.Println(err)
		return JSON(http.StatusInternalServerError, Errors("Can't connect to database"))
	default:
		http.Redirect(w, r, "/mtg/cards/"+card, http.StatusFound)
		return JSON(http.StatusFound, []string{"Redirecting to /mtg/cards/" + card})
	}
}

func HandleCard(db *sql.DB, pl *PriceList, params martini.Params) (int, []byte) {
	card, err := FetchCard(db, params["id"])
	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Card not found"))
	}
	card.Fill(pl)
	return JSON(http.StatusOK, card)
}

func HandleSets(db *sql.DB) (int, []byte) {
	sets, err := FetchSets(db)
	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Sets not found"))
	}
	return JSON(http.StatusOK, sets)
}

func HandleSet(db *sql.DB, params martini.Params) (int, []byte) {
	card, err := FetchSet(db, params["id"])

	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors("Set not found"))
	}

	return JSON(http.StatusOK, card)
}

func HandleTerm(term string) func(*sql.DB) (int, []byte) {
	return func(db *sql.DB) (int, []byte) {
		terms, err := FetchTerms(db, term)
		if err != nil {
			log.Println(err)
			return JSON(http.StatusNotFound, Errors(term+" not found"))
		}
		return JSON(http.StatusOK, terms)
	}
}

func HandleTypeahead(db *sql.DB, pl *PriceList, req *http.Request) (int, []byte) {
	cards, err := FetchTypeahead(db, req.URL.Query().Get("q"))
	if err != nil {
		log.Println(err)
		return JSON(http.StatusNotFound, Errors(" Can't find any cards that match that search"))
	}
	for i, _ := range cards {
		cards[i].Fill(pl)
	}
	return JSON(http.StatusOK, cards)
}

type Pong struct {
	Rally string `json:"rally"`
}

func Ping(db *sql.DB) (int, []byte) {
	if db.Ping() != nil {
		return JSON(http.StatusInternalServerError, Errors("The database could not be reached"))
	}
	return JSON(http.StatusOK, Pong{Rally: "serve"})
}

func NotFound() (int, []byte) {
	return JSON(http.StatusNotFound, Errors("No endpoint here"))
}

func NewApi() *martini.Martini {
	m := martini.New()

	// Setup middleware
	m.Use(martini.Recovery())
	m.Use(martini.Logger())
	m.Use(gzip.All())
	m.Use(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/mtg/cards/random" {
			w.Header().Set("Cache-Control", "public,max-age=3600")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "link,content-length")
		w.Header().Set("License", "The textual information presented through this API about Magic: The Gathering is copyrighted by Wizards of the Coast.")
		w.Header().Set("Disclaimer", "This API is not produced, endorsed, supported, or affiliated with Wizards of the Coast.")
		w.Header().Set("Pricing", "store.tcgplayer.com allows you to buy cards from any of our vendors, all at the same time, in a simple checkout experience. Shop, Compare & Save with TCGplayer.com!")
		w.Header().Set("Strict-Transport-Security", "max-age=86400")
	})

	r := martini.NewRouter()

	r.Get("/mtg/cards", HandleCards)
	r.Get("/mtg/cards/typeahead", HandleTypeahead)
	r.Get("/mtg/cards/random", HandleRandomCard)
	r.Get("/mtg/cards/:id", HandleCard)
	r.Get("/mtg/sets", HandleSets)
	r.Get("/mtg/sets/:id", HandleSet)
	r.Get("/mtg/colors", HandleTerm("colors"))
	r.Get("/mtg/supertypes", HandleTerm("supertypes"))
	r.Get("/mtg/subtypes", HandleTerm("subtypes"))
	r.Get("/mtg/types", HandleTerm("types"))
	r.NotFound(NotFound)

	m.Action(r.Handle)
	return m
}

func updatePrices(db *sql.DB, pl *PriceList) {
	for {
		time.Sleep(30 * time.Minute)
		log.Println("Fetching new prices")
		prices, err := loadPrices(db)
		if err != nil {
			log.Println(err)
		}
		pl.Prices = prices
	}
}

func ServeWebsite() error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	prices, err := FetchPrices(db)
	if err != nil {
		return err
	}
	pricelist := PriceList{}
	pricelist.Prices = prices

	m := NewApi()
	m.Map(db)
	m.Map(&pricelist)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	m.RunOnAddr(":" + port)
	return nil
}
