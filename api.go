package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"stackmachine.com/cql"

	"golang.org/x/net/context"

	_ "github.com/lib/pq"
	"goji.io"
	"goji.io/pat"
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

func (c *Card) Fill() {
	c.Href = ReverseCard(c.Id)
	c.StoreUrl = TCGCardURL(c)

	for i, _ := range c.Editions {
		e := &c.Editions[i]
		e.Href = ReverseEdition(e.MultiverseId)
		e.SetUrl = ReverseSet(e.SetId)
		e.ImageUrl = MTGImageURL(e.MultiverseId)
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

func (s *Set) Fill() {
	s.Href = fmt.Sprintf("%s/mtg/sets/%s", GetHostname(), s.Id)
	s.CardsUrl = fmt.Sprintf("%s/mtg/cards?set=%s", GetHostname(), s.Id)
}

func JSON(w http.ResponseWriter, code int, val interface{}) {
	blob, err := json.MarshalIndent(val, "", "  ")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "Internal server error :("}"`)
	} else {
		w.WriteHeader(code)
		fmt.Fprintf(w, string(blob))
	}
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

type API struct {
	db *cql.DB
}

func (a *API) HandleCards(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cond, err, errors := ParseSearch(r.URL)
	if err != nil {
		JSON(w, http.StatusBadRequest, Errors(errors...))
		return
	}
	page, err := CardsPaging(r.URL)
	if err != nil {
		JSON(w, http.StatusBadRequest, Errors(errors...))
		return
	}
	cards, err := FetchCards(a.db, cond, page)
	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Cards not found"))
		return
	}

	for i, _ := range cards {
		cards[i].Fill()
	}

	w.Header().Set("Link", LinkHeader(GetHostname(), r.URL, page))
	JSON(w, http.StatusOK, cards)
}

func (a *API) HandleRandomCard(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var card string
	err := a.db.QueryRow("SELECT id FROM cards ORDER BY RANDOM() LIMIT 1").Scan(&card)
	switch {
	case err == sql.ErrNoRows:
		JSON(w, http.StatusNotFound, Errors("No random card can be found"))
	case err != nil:
		JSON(w, http.StatusInternalServerError, Errors("Can't connect to database"))
	default:
		http.Redirect(w, r, "/mtg/cards/"+card, http.StatusFound)
		JSON(w, http.StatusFound, []string{"Redirecting to /mtg/cards/" + card})
	}
}

func (a *API) HandleCard(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	card, err := FetchCard(a.db, pat.Param(ctx, "id"))
	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Card not found"))
		return
	}

	card.Fill()

	JSON(w, http.StatusOK, card)
}

func (a *API) HandleSets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sets, err := FetchSets(a.db)
	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Sets not found"))
	} else {
		JSON(w, http.StatusOK, sets)
	}
}

func (a *API) HandleSet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	card, err := FetchSet(ctx, a.db, pat.Param(ctx, "id"))

	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Set not found"))
	} else {
		JSON(w, http.StatusOK, card)
	}
}

func (a *API) HandleTerm(term string) func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		terms, err := FetchTerms(a.db, term)
		if err != nil {
			JSON(w, http.StatusNotFound, Errors(term+" not found"))
		} else {
			JSON(w, http.StatusOK, terms)
		}
	}
}

func (a *API) HandleTypeahead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cards, err := FetchTypeahead(a.db, r.URL.Query().Get("q"))
	if err != nil {
		JSON(w, http.StatusNotFound, Errors(" Can't find any cards that match that search"))
		return
	}

	for i, _ := range cards {
		cards[i].Fill()
	}

	JSON(w, http.StatusOK, cards)
}

func NotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusNotFound, Errors("No endpoint here"))
}

func NewAPI(db *cql.DB) (http.Handler, error) {
	mux := goji.NewMux()
	app := API{db: db}

	// Setup middleware
	// mux.UseC(Tracing)
	mux.UseC(Headers)

	mux.HandleFuncC(pat.Get("/mtg/cards"), app.HandleCards)
	mux.HandleFuncC(pat.Get("/mtg/cards/typeahead"), app.HandleTypeahead)
	mux.HandleFuncC(pat.Get("/mtg/cards/random"), app.HandleRandomCard)
	mux.HandleFuncC(pat.Get("/mtg/cards/:id"), app.HandleCard)
	mux.HandleFuncC(pat.Get("/mtg/sets"), app.HandleSets)
	mux.HandleFuncC(pat.Get("/mtg/sets/:id"), app.HandleSet)
	mux.HandleFuncC(pat.Get("/mtg/colors"), app.HandleTerm("colors"))
	mux.HandleFuncC(pat.Get("/mtg/supertypes"), app.HandleTerm("supertypes"))
	mux.HandleFuncC(pat.Get("/mtg/subtypes"), app.HandleTerm("subtypes"))
	mux.HandleFuncC(pat.Get("/mtg/types"), app.HandleTerm("types"))

	//r.NotFound(NotFound)
	return mux, nil
}

func ServeWebsite() error {
	db, err := getDatabase()
	if err != nil {
		return err
	}

	m, err := NewAPI(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.ListenAndServe(":"+port, m)
	return nil
}
