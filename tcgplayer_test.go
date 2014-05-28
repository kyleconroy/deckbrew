package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"testing"
)

func TestPrice(t *testing.T) {
	if parseMoney("$2,799.00") != 279900 {
		t.Errorf("Couldn't not parse money with dollar sign and comma")
	}
}
