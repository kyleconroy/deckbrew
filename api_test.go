package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	if Slug(name) != "æther-adept" {
		t.Errorf("%s != æther-adept", Slug(name))
	}
}

func TestApi(t *testing.T) {
	db, err := GetDatabase()
	pricelist := PriceList{}

	if err != nil {
		t.Fatal(err)
	}

	m := NewApi()
	m.Map(db)
	m.Map(&pricelist)

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

	// Test Random
	res, err := http.Get(ts.URL + "/mtg/cards/random")

	if err != nil {
		t.Error(err)
	}

	if res.Request.URL.String() == "/mtg/cards/random" {
		t.Errorf("Expected /mtg/cards/random redirect to a new page")
	}

	loadFirstCard := func(u string) (Card, error) {
		var card Card

		res, err := http.Get(ts.URL + u)

		if err != nil {
			return card, err
		}

		if res.StatusCode != 200 {
			return card, fmt.Errorf("Expected %s to return 200, not %d", u, res.StatusCode)
		}

		defer res.Body.Close()

		blob, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return card, err
		}

		var cards []Card

		err = json.Unmarshal(blob, &cards)

		if err != nil {
			return card, err
		}

		return cards[0], nil
	}

	// Test Paging
	pageone, _ := loadFirstCard("/mtg/cards?page=1")
	pagetwo, _ := loadFirstCard("/mtg/cards?page=2")

	if pageone.Id == pagetwo.Id {
		t.Errorf("Page one and two both have the same first card, %s", pageone.Id)
	}
}
