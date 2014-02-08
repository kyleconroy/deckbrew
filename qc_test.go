package main

import (
	"testing"
)

func TestSelect(t *testing.T) {
        users := Select("*").From("users")
        active := users.Where(And(Eq("foo", 0), Eq("bar", 1)))
        window := active.Limit(1).Offset(1)

        sql, args, err := window.ToSql()

        if err != nil {
                t.Fatal(err)
        }

        if sql != "SELECT * FROM users WHERE (foo = $1 AND bar = $2) LIMIT 1 OFFSET 1" {
                t.Errorf("Malformed SQL: %s", sql)
        }

        if len(args) > 0 && args[0].(int) != 0 {
                t.Fatalf("Incorrect args %+V", args)
        }
}
