package api

import (
	"testing"
)

func TestLoadCardJSON(t *testing.T) {
	collection, err := LoadCollection("cards.json")

	if err != nil {
		t.Fatal(err)
	}

	_, ok := collection["LEA"]

	if !ok {
		t.Fatal("The collection did not load properly")
	}
}
