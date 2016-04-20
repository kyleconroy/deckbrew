package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/kyleconroy/deckbrew/config"
	"github.com/kyleconroy/migrator"
	_ "github.com/lib/pq"
	"stackmachine.com/cql"
)

func FetchSet(ctx context.Context, db *cql.DB, id string) (Set, error) {
	var set Set
	row := db.QueryRowC(ctx, "SELECT id,name,border,type FROM sets WHERE id = $1", id)
	err := row.Scan(&set.Id, &set.Name, &set.Border, &set.Type)
	return set, err
}

func FetchSets(ctx context.Context, db *cql.DB) ([]Set, error) {
	sets := []Set{}
	rows, err := db.QueryC(ctx, "SELECT id,name,border,type,price_guide,priced FROM sets ORDER BY name")
	if err != nil {
		return sets, err
	}
	defer rows.Close()
	for rows.Next() {
		var set Set
		if err := rows.Scan(&set.Id, &set.Name, &set.Border, &set.Type, &set.PriceGuide, &set.Priced); err != nil {
			return sets, err
		}
		set.Fill()
		sets = append(sets, set)
	}
	return sets, nil
}

func FetchTerms(ctx context.Context, db *cql.DB, term string) ([]string, error) {
	result := []string{}

	rows, err := db.QueryC(ctx, "select distinct unnest("+term+") as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var term string

		if err := rows.Scan(&term); err != nil {
			return result, err
		}

		result = append(result, term)
	}

	return result, rows.Err()
}

func scanCards(rows *sql.Rows) ([]Card, error) {
	cards := []Card{}

	defer rows.Close()

	for rows.Next() {
		var blob []byte
		var card Card

		if err := rows.Scan(&blob); err != nil {
			return cards, err
		}

		err := json.Unmarshal(blob, &card)

		if err != nil {
			return cards, err
		}

		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return cards, err
	}

	return cards, nil
}

func FetchTypeahead(ctx context.Context, db *cql.DB, search string) ([]Card, error) {
	if strings.ContainsAny(search, "%_") {
		return []Card{}, fmt.Errorf("Search string can't contain '%%' or '_'")
	}

	rows, err := db.QueryC(ctx, "SELECT record FROM cards WHERE name ILIKE $1 ORDER BY name LIMIT 10", search+"%")

	if err != nil {
		return []Card{}, err
	}

	return scanCards(rows)
}

func FetchCards(ctx context.Context, db *cql.DB, cond Condition, page int) ([]Card, error) {
	query := Select("record").From("cards").Where(cond).OrderBy("name", true)
	limit := query.Limit(100).Offset(page * 100)

	ql, items, err := limit.ToSql()

	if err != nil {
		return []Card{}, err
	}

	rows, err := db.QueryC(ctx, ql, items...)

	if err != nil {
		return []Card{}, err
	}

	return scanCards(rows)
}

func FetchCardIDs(ctx context.Context, db *cql.DB) ([]string, error) {
	ids := []string{}

	rows, err := db.QueryC(ctx, "SELECT id FROM cards")
	if err != nil {
		return ids, err
	}

	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func FetchCard(ctx context.Context, db *cql.DB, id string) (Card, error) {
	var blob []byte
	var card Card
	err := db.QueryRowC(ctx, "SELECT record FROM cards WHERE id = $1", id).Scan(&blob)
	if err == sql.ErrNoRows {
		return card, fmt.Errorf("No card with ID %s", id)
	}
	if err != nil {
		return card, err
	}
	return card, json.Unmarshal(blob, &card)
}

func FetchPrices(ctx context.Context, db *cql.DB) (map[string]Price, error) {
	prices := map[string]Price{}

	rows, err := db.QueryC(ctx, `
    SELECT DISTINCT ON (multiverse_id) multiverse_id, low, high, median
    FROM prices
    ORDER BY multiverse_id, created DESC
    `)

	if err != nil {
		return prices, err
	}

	defer rows.Close()
	for rows.Next() {
		var id string
		var price Price
		if err := rows.Scan(&id, &price.Low, &price.High, &price.Average); err != nil {
			return prices, err
		}
		prices[id] = price
	}
	return prices, rows.Err()
}

func InsertPrice(ctx context.Context, db *cql.DB, id string, price Price) error {
	_, err := db.ExecC(ctx, `
    INSERT INTO prices (multiverse_id, low, high, median)
    VALUES ($1, $2, $3, $4)
    `, id, price.Low, price.High, price.Average)
	return err
}

func MigrateDatabase() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	return migrator.Run(cfg.DB.DB, "migrations")
}
