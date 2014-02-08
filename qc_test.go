package main

import (
	"testing"
)

func TestSelect(t *testing.T) {
        users := Select("*").From("users")
        active := users.Where(Eq("foo", 0))

        sql, args, err := active.ToSql()

        if err != nil {
                t.Fatal(err)
        }

        if sql != "SELECT * FROM users WHERE foo = $1" {
                t.Errorf("Malformed SQL: %s", sql)
        }

        if len(args) > 0 && args[0].(int) != 0 {
                t.Fatalf("Incorrect args %+V", args)
        }
}
