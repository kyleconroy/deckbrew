package main

import (
	"net/url"
	"testing"
)

func TestLinkHeader(t *testing.T) {
	url, _ := url.Parse("/cards?foo=bar")
    header := LinkHeader("http", "localhost:3000", url, Query{Page: 0})
	expected := "<http://localhost:3000/cards?foo=bar&page=1>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

	url, _ = url.Parse("/cards?foo=bar&page=1")
    header = LinkHeader("http", "localhost:3000", url, Query{Page: 1})
	expected = "<http://localhost:3000/cards?foo=bar&page=0>; rel=\"prev\", <http://localhost:3000/cards?foo=bar&page=2>; rel=\"next\""

	if header != expected {
		t.Errorf("Expected %s not %s", expected, header)
	}

}
