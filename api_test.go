package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLinkHeader(t *testing.T) {
	url, _ := url.Parse("/cards?foo=bar")
	header := LinkHeader("http://localhost:3000", url, 0)
	expected := "<http://localhost:3000/cards?foo=bar&page=1>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

	url, _ = url.Parse("/cards?foo=bar&page=1")
	header = LinkHeader("http://localhost:3000", url, 1)
	expected = "<http://localhost:3000/cards?foo=bar&page=0>; rel=\"prev\", <http://localhost:3000/cards?foo=bar&page=2>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

}

func TestSlug(t *testing.T) {
	name := "Æther Adept?.\"':,"

	if Slug(name) != "aether-adept" {
		t.Errorf("%s != aether-adept", Slug(name))
	}
}

func TestApi(t *testing.T) {
	db, err := GetDatabase()

	if err != nil {
		t.Fatal(err)
	}

	m := NewApi()
	m.Map(db)

	ts := httptest.NewServer(m)
	defer ts.Close()

	urls := []string{
		"/mtg/cards",
		"/mtg/cards?type=creature",
		"/mtg/cards?subtype=zombie",
		"/mtg/cards?supertype=legendary",
		"/mtg/cards?color=red",
		"/mtg/cards?name=rats",
		"/mtg/cards?set=UNH",
		"/mtg/cards?rarity=mythic",
		"/mtg/cards?rarity=basic",
		"/mtg/cards?oracle=you+win+the+game",
		"/mtg/cards/time-vault",
		"/mtg/cards/typeahead?q=nessian",
		"/mtg/sets",
		"/mtg/sets/UNH",
		"/mtg/colors",
		"/mtg/types",
		"/mtg/supertypes",
		"/mtg/subtypes",
	}

	for _, url := range urls {

		res, err := http.Get(ts.URL + url)

		if err != nil {
			t.Error(err)
		}

		if res.StatusCode != 200 {
			t.Errorf("Expected %s to return 200, not %d", url, res.StatusCode)
		}
	}
}
