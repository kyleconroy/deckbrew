package brew

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/kyleconroy/deckbrew/config"

	"golang.org/x/net/context"
	"stackmachine.com/cql"
)

const querySet = `
SELECT id, name, border, type FROM sets WHERE id = $1
`

const querySets = `
SELECT id, name, border, type, price_guide, priced FROM sets ORDER BY name
`

const queryTypeahead = `
SELECT record FROM cards WHERE name ILIKE $1 ORDER BY name LIMIT 10
`

const queryCard = `
SELECT record FROM cards WHERE id = $1
`

const queryColors = `
SELECT DISTINCT unnest(colors) as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC
`

const queryTypes = `
SELECT DISTINCT unnest(types) as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC
`

const querySupertypes = `
SELECT DISTINCT unnest(supertypes) as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC
`

const querySubtypes = `
SELECT DISTINCT unnest(subtypes) as t from cards WHERE NOT sets && '{unh,ugl}' ORDER BY t ASC
`

const queryRandomCard = `
SELECT id FROM cards ORDER BY RANDOM() LIMIT 1
`

const queryCards = `
SELECT record FROM cards
WHERE
  ($1 OR multicolor = $2) AND
  ($3 OR rarities && $4) AND
  ($5 OR types && $6) AND 
  ($7 OR supertypes && $8) AND 
  ($9 OR colors && $10) AND 
  ($11 OR subtypes && $12) AND 
  ($13 OR formats && $14) AND 
  ($15 OR status && $16) AND 
  ($17 OR mids && $18) AND 
  ($19 OR sets && $20) AND 
  ($21 OR name ILIKE ANY ($22)) AND
  ($23 OR rules ILIKE ANY ($24))
ORDER BY name ASC
LIMIT $25
OFFSET $26
`

type client struct {
	db     *cql.DB
	router router

	// Prepared statements
	stmtGetSet        *cql.Stmt
	stmtGetSets       *cql.Stmt
	stmtTypeahead     *cql.Stmt
	stmtGetCard       *cql.Stmt
	stmtGetCards      *cql.Stmt
	stmtGetTypes      *cql.Stmt
	stmtGetSubtypes   *cql.Stmt
	stmtGetSupertypes *cql.Stmt
	stmtGetColors     *cql.Stmt
	stmtRandomCard    *cql.Stmt
}

func NewReader(cfg *config.Config) (Reader, error) {
	c := &client{db: cfg.DB, router: router{cfg}}
	var err error

	for _, pair := range []struct {
		stmt  **cql.Stmt
		query string
	}{
		{&c.stmtGetSet, querySet},
		{&c.stmtGetSets, querySets},
		{&c.stmtGetCard, queryCard},
		{&c.stmtGetCards, queryCards},
		{&c.stmtTypeahead, queryTypeahead},
		{&c.stmtGetColors, queryColors},
		{&c.stmtGetTypes, queryTypes},
		{&c.stmtGetSupertypes, querySupertypes},
		{&c.stmtGetSubtypes, querySubtypes},
		{&c.stmtRandomCard, queryRandomCard},
	} {
		*pair.stmt, err = c.db.PrepareC(context.TODO(), pair.query)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *client) GetSet(ctx context.Context, id string) (Set, error) {
	var set Set
	row := c.stmtGetSet.QueryRowC(ctx, id)
	err := row.Scan(&set.Id, &set.Name, &set.Border, &set.Type)
	return set, err
}

func (c *client) GetSets(ctx context.Context) ([]Set, error) {
	sets := []Set{}
	rows, err := c.stmtGetSets.QueryC(ctx)
	if err != nil {
		return sets, err
	}
	defer rows.Close()
	for rows.Next() {
		var set Set
		if err := rows.Scan(&set.Id, &set.Name, &set.Border, &set.Type, &set.PriceGuide, &set.Priced); err != nil {
			return sets, err
		}
		set.Fill(c.router)
		sets = append(sets, set)
	}
	return sets, nil
}

func scanCards(rows *sql.Rows, r router) ([]Card, error) {
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
	for i, _ := range cards {
		cards[i].Fill(r)
	}
	return cards, nil
}

func (c *client) GetCardsByName(ctx context.Context, search string) ([]Card, error) {
	if strings.ContainsAny(search, "%_") {
		return []Card{}, fmt.Errorf("Search string can't contain '%%' or '_'")
	}

	rows, err := c.stmtTypeahead.QueryC(ctx, search+"%")
	if err == sql.ErrNoRows {
		return []Card{}, nil
	}
	if err != nil {
		return []Card{}, err
	}

	return scanCards(rows, c.router)
}

func sarray(values []string) string {
	return "{" + strings.Join(values, ",") + "}"
}

func (c *client) GetCards(ctx context.Context, s Search) ([]Card, error) {
	rows, err := c.stmtGetCards.QueryC(ctx,
		// Multicolor, ignored by default
		!s.IncludeMulticolor, s.Multicolor,
		len(s.Rarities) == 0, sarray(s.Rarities),
		len(s.Types) == 0, sarray(s.Types),
		len(s.Supertypes) == 0, sarray(s.Supertypes),
		len(s.Colors) == 0, sarray(s.Colors),
		len(s.Subtypes) == 0, sarray(s.Subtypes),
		len(s.Formats) == 0, sarray(s.Formats),
		len(s.Status) == 0, sarray(s.Status),
		len(s.MultiverseIDs) == 0, sarray(s.MultiverseIDs),
		len(s.Sets) == 0, sarray(s.Sets),
		len(s.Names) == 0, sarray(s.Names),
		len(s.Rules) == 0, sarray(s.Rules),
		s.Limit, s.Offset,
	)
	if err == sql.ErrNoRows {
		return []Card{}, nil
	}
	if err != nil {
		log.Println(err)
		return []Card{}, err
	}

	return scanCards(rows, c.router)
}

func (c *client) GetRandomCardID(ctx context.Context) (string, error) {
	var id string
	err := c.stmtRandomCard.QueryRowC(ctx).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (c *client) GetCard(ctx context.Context, id string) (Card, error) {
	var blob []byte
	var card Card
	err := c.stmtGetCard.QueryRowC(ctx, id).Scan(&blob)
	if err == sql.ErrNoRows {
		return card, fmt.Errorf("No card with ID %s", id)
	}
	if err != nil {
		return card, err
	}
	if err := json.Unmarshal(blob, &card); err != nil {
		return card, err
	}
	card.Fill(c.router)
	return card, nil
}

func (c *client) fetchTerms(ctx context.Context, stmt *cql.Stmt) ([]string, error) {
	result := []string{}

	rows, err := stmt.QueryC(ctx)
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

func (c *client) GetColors(ctx context.Context) ([]string, error) {
	return c.fetchTerms(ctx, c.stmtGetColors)
}

func (c *client) GetSupertypes(ctx context.Context) ([]string, error) {
	return c.fetchTerms(ctx, c.stmtGetSupertypes)
}

func (c *client) GetSubtypes(ctx context.Context) ([]string, error) {
	return c.fetchTerms(ctx, c.stmtGetSubtypes)
}

func (c *client) GetTypes(ctx context.Context) ([]string, error) {
	return c.fetchTerms(ctx, c.stmtGetTypes)
}
