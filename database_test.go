package main

import (
	"testing"
)

func TestBannedRestricted(t *testing.T) {
	funny := Card{Sets: []string{"unh"}}
	fake := Card{Sets: []string{"ugl"}}
	legal := Card{Sets: []string{"m14"}}
	banned := Card{Id: "banned"}
	restricted := Card{Id: "restricted"}

	format := Format{
		Sets:       []string{"m14"},
		Banned:     []Card{banned},
		Restricted: []Card{restricted},
	}

	if format.CardStatus(&funny) != 0 {
		t.Error("An unhinged card should always be illegal")
	}

	if format.CardStatus(&fake) != 0 {
		t.Error("An unglued card should always be illegal")
	}

	if format.CardStatus(&legal) != 1 {
		t.Error("An M14 card should legal")

	}

	if format.CardStatus(&banned) != 3 {
		t.Error("This card should be banned")
	}

	if format.CardStatus(&restricted) != 2 {
		t.Error("This card should be restricted")
	}
}
