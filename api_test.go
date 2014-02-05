package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLinkHeader(t *testing.T) {
	url, _ := url.Parse("/cards?foo=bar")
	header := LinkHeader("http://localhost:3000", url, Query{Page: 0})
	expected := "<http://localhost:3000/cards?foo=bar&page=1>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

	url, _ = url.Parse("/cards?foo=bar&page=1")
	header = LinkHeader("http://localhost:3000", url, Query{Page: 1})
	expected = "<http://localhost:3000/cards?foo=bar&page=0>; rel=\"prev\", <http://localhost:3000/cards?foo=bar&page=2>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

}

func TestApi(t *testing.T) {
	db, err := Open("postgres://urza:power9@localhost/deckbrew?sslmode=disable")

	if err != nil {
		t.Fatal(err)
	}

	m := NewApi(&db)

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
		"/mtg/cards/1cdf2b87355ed978c0c5fe64bfc6a38c",
		"/mtg/editions/73935",
		"/mtg/sets",
		"/mtg/sets/UNH",
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
