package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/kyleconroy/deckbrew/brew"
	"github.com/kyleconroy/deckbrew/config"
	_ "github.com/lib/pq"

	"goji.io"
	"goji.io/pat"
)

// XXX: Use the config instead
func Slug(name string) string {
	re := regexp.MustCompile(`[,.'"?:()]`)
	d := strings.Replace(strings.ToLower(name), " ", "-", -1)
	return re.ReplaceAllLiteralString(d, "")
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
	c    brew.Reader
	host string
}

func (a *API) apiBase() string {
	if strings.Contains(a.host, ":") {
		return "http://" + a.host
	} else {
		return "https://" + a.host
	}
}

func (a *API) HandleCards(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	s, err, errors := ParseSearch(r.URL)
	if err != nil {
		JSON(w, http.StatusBadRequest, Errors(errors...))
		return
	}
	page, err := CardsPaging(r.URL)
	if err != nil {
		JSON(w, http.StatusBadRequest, Errors(errors...))
		return
	}
	cards, err := a.c.GetCards(ctx, s, page)
	if err != nil {
		JSON(w, http.StatusInternalServerError, Errors("Error fetching cards"))
		return
	}
	w.Header().Set("Link", LinkHeader(a.apiBase(), r.URL, page))
	JSON(w, http.StatusOK, cards)
}

func (a *API) HandleRandomCard(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := a.c.GetRandomCardID(ctx)
	switch {
	case id == "":
		JSON(w, http.StatusNotFound, Errors("No random card can be found"))
	case err != nil:
		JSON(w, http.StatusInternalServerError, Errors("Can't connect to database"))
	default:
		http.Redirect(w, r, "/mtg/cards/"+id, http.StatusFound)
		fmt.Fprintf(w, "[Redirecting to /mtg/cards/golden-wish]")
	}
}

func (a *API) HandleCard(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	card, err := a.c.GetCard(ctx, pat.Param(ctx, "id"))
	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Card not found"))
		return
	}
	JSON(w, http.StatusOK, card)
}

func (a *API) HandleSets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sets, err := a.c.GetSets(ctx)
	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Sets not found"))
	} else {
		JSON(w, http.StatusOK, sets)
	}
}

func (a *API) HandleSet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	card, err := a.c.GetSet(ctx, pat.Param(ctx, "id"))

	if err != nil {
		JSON(w, http.StatusNotFound, Errors("Set not found"))
	} else {
		JSON(w, http.StatusOK, card)
	}
}

func (a *API) HandleTerm(f func(context.Context) ([]string, error)) func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		terms, err := f(ctx)
		if err != nil {
			JSON(w, http.StatusNotFound, Errors("no strings found"))
		} else {
			JSON(w, http.StatusOK, terms)
		}
	}
}

func (a *API) HandleTypeahead(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	cards, err := a.c.GetCardsByName(ctx, r.URL.Query().Get("q"))
	if err != nil {
		JSON(w, http.StatusNotFound, Errors(" Can't find any cards that match that search"))
		return
	}
	JSON(w, http.StatusOK, cards)
}

func NotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusNotFound, Errors("No endpoint here"))
}

type term int

func New(cfg *config.Config, client brew.Reader) http.Handler {
	app := API{c: client, host: cfg.HostAPI}

	mux := goji.NewMux()

	// Setup middleware
	mux.UseC(Recover)
	mux.UseC(Tracing)
	mux.UseC(Headers)
	mux.UseC(Recover)

	mux.HandleFuncC(pat.Get("/mtg/cards"), app.HandleCards)
	mux.HandleFuncC(pat.Get("/mtg/cards/typeahead"), app.HandleTypeahead)
	mux.HandleFuncC(pat.Get("/mtg/cards/random"), app.HandleRandomCard)
	mux.HandleFuncC(pat.Get("/mtg/cards/:id"), app.HandleCard)
	mux.HandleFuncC(pat.Get("/mtg/sets"), app.HandleSets)
	mux.HandleFuncC(pat.Get("/mtg/sets/:id"), app.HandleSet)
	mux.HandleFuncC(pat.Get("/mtg/colors"), app.HandleTerm(client.GetColors))
	mux.HandleFuncC(pat.Get("/mtg/supertypes"), app.HandleTerm(client.GetSupertypes))
	mux.HandleFuncC(pat.Get("/mtg/subtypes"), app.HandleTerm(client.GetSubtypes))
	mux.HandleFuncC(pat.Get("/mtg/types"), app.HandleTerm(client.GetTypes))

	return mux
}
