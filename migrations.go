package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

func MigrateDatabase() error {
	db, err := getDatabase()
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS migrations (version integer DEFAULT 0)`)
	if err != nil {
		return err
	}

	migrations := [][]string{
		createIntitalTables(),
	}

	for i, migration := range migrations {
		var s int
		schema := i + 1
		err := db.QueryRow("SELECT version FROM migrations WHERE version=$1", schema).Scan(&s)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if s == schema {
			log.Printf("migration %d has already run\n", s)
			return nil
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, query := range migration {
			log.Println(query)
			_, err = tx.Exec(query)
			if err != nil {
				log.Println(err)
				return tx.Rollback()
			}
		}
		_, err = tx.Exec("INSERT INTO migrations (version) VALUES ($1)", schema)
		if err != nil {
			log.Println(err)
			return tx.Rollback()
		}
		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

func createIntitalTables() []string {
	return []string{
		"CREATE EXTENSION pg_trgm",

		`CREATE TABLE cards (
        id                varchar(150)   primary key,
        name              varchar(200)   DEFAULT '',
        mana_cost         varchar(45)    DEFAULT '',
        toughness         varchar(6)     DEFAULT '',
        power             varchar(6)     DEFAULT '',
        types             varchar(20)[]  DEFAULT '{}',
        rarities          varchar(15)[]  DEFAULT '{}',
        sets              varchar(3)[]   DEFAULT '{}',
        subtypes          varchar(100)[] DEFAULT '{}',
        supertypes        varchar(100)[] DEFAULT '{}',
        colors            varchar(5)[]   DEFAULT '{}',
        formats           varchar(9)[]   DEFAULT '{}',
        status            varchar(10)[]  DEFAULT '{}',
        mids              varchar(20)[]  DEFAULT '{}',
        multicolor        boolean        DEFAULT false,
        record            text           DEFAULT '',
        rules             text           DEFAULT '',
        loyalty           integer        DEFAULT 0,
        cmc               integer        DEFAULT 0)`,

		`CREATE TABLE prices (
        id                integer        primary key,
        card_id           varchar(150)   DEFAULT '',
        name              varchar(200)   DEFAULT '',
        rarity            varchar(15)    DEFAULT '',
        set               varchar(3)     DEFAULT '',
        foil              boolean        DEFAULT false,
        price_low         integer        DEFAULT 0,
        price_median      integer        DEFAULT 0,
        price_high        integer        DEFAULT 0)`,

		`CREATE TABLE sets (
        id                varchar(3) primary key,
        name              varchar(200) DEFAULT '',
        border            varchar(40) DEFAULT '',
        type              varchar(32) DEFAULT '')`,

		"CREATE INDEX cards_name_id ON cards(id)",
		"CREATE INDEX cards_power_index ON cards(power)",
		"CREATE INDEX cards_toughness_index ON cards(toughness)",
		"CREATE INDEX cards_names_sort_index ON cards(name)",
		"CREATE INDEX cards_multicolor_index ON cards(multicolor)",
		"CREATE INDEX cards_types_index ON cards USING GIN(types)",
		"CREATE INDEX cards_subtypes_index ON cards USING GIN(subtypes)",
		"CREATE INDEX cards_supertypes_index ON cards USING GIN(supertypes)",
		"CREATE INDEX cards_colors_index ON cards USING GIN(colors)",
		"CREATE INDEX cards_sets_index ON cards USING GIN(sets)",
		"CREATE INDEX cards_rarities_index ON cards USING GIN(rarities)",
		"CREATE INDEX cards_status_index ON cards USING GIN(status)",
		"CREATE INDEX cards_formats_index ON cards USING GIN(formats)",
		"CREATE INDEX cards_mids_index ON cards USING GIN(mids)",
		"CREATE INDEX cards_names_index ON cards USING GIN(name gin_trgm_ops)",
		"CREATE INDEX cards_rules_index ON cards USING GIN(rules gin_trgm_ops)",
		"CREATE INDEX prices_id_index ON prices(id)",
		"CREATE INDEX prices_card_index ON prices(card_id)",
		"CREATE INDEX prices_set_index ON prices(set)",
		"CREATE INDEX prices_rarity_index ON prices(rarity)",
		"CREATE INDEX prices_low_index ON prices(price_low)",
		"CREATE INDEX prices_median_index ON prices(price_median)",
		"CREATE INDEX prices_high_index ON prices(price_high)",
		"CREATE INDEX prices_names_trgm ON prices USING GIN(name gin_trgm_ops)",
		"CREATE INDEX sets_type_index ON sets(type)",
		"CREATE INDEX sets_border_index ON sets(border)",
	}
}
