package main

import (
	"database/sql"
	_ "github.com/lib/pq"
)

func Migrate(db *sql.DB) error {
	exec(db, "CREATE EXTENSION pg_trgm")

	exec(db, `CREATE TABLE cards (
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
        cmc               integer        DEFAULT 0)`)

	exec(db, `CREATE TABLE prices (
        id                integer        primary key,
        card_id           varchar(150)   DEFAULT '',
        name              varchar(200)   DEFAULT '',
        rarity            varchar(15)    DEFAULT '',
        set               varchar(3)     DEFAULT '',
        foil              boolean        DEFAULT false,
        price_low         integer        DEFAULT 0,
        price_median      integer        DEFAULT 0,
        price_high        integer        DEFAULT 0)`)

	exec(db, `CREATE TABLE sets (
        id                varchar(3) primary key,
        name              varchar(200) DEFAULT '',
        border            varchar(40) DEFAULT '',
        type              varchar(32) DEFAULT '')`)

	exec(db, "CREATE INDEX cards_name_id ON cards(id)")
	exec(db, "CREATE INDEX cards_power_index ON cards(power)")
	exec(db, "CREATE INDEX cards_toughness_index ON cards(toughness)")
	exec(db, "CREATE INDEX cards_names_sort_index ON cards(name)")
	exec(db, "CREATE INDEX cards_multicolor_index ON cards(multicolor)")
	exec(db, "CREATE INDEX cards_types_index ON cards USING GIN(types)")
	exec(db, "CREATE INDEX cards_subtypes_index ON cards USING GIN(subtypes)")
	exec(db, "CREATE INDEX cards_supertypes_index ON cards USING GIN(supertypes)")
	exec(db, "CREATE INDEX cards_colors_index ON cards USING GIN(colors)")
	exec(db, "CREATE INDEX cards_sets_index ON cards USING GIN(sets)")
	exec(db, "CREATE INDEX cards_rarities_index ON cards USING GIN(rarities)")
	exec(db, "CREATE INDEX cards_status_index ON cards USING GIN(status)")
	exec(db, "CREATE INDEX cards_formats_index ON cards USING GIN(formats)")
	exec(db, "CREATE INDEX cards_mids_index ON cards USING GIN(mids)")
	exec(db, "CREATE INDEX cards_names_index ON cards USING GIN(name gin_trgm_ops)")
	exec(db, "CREATE INDEX cards_rules_index ON cards USING GIN(rules gin_trgm_ops)")

	exec(db, "CREATE INDEX prices_id_index ON prices(id)")
	exec(db, "CREATE INDEX prices_card_index ON prices(card_id)")
	exec(db, "CREATE INDEX prices_set_index ON prices(set)")
	exec(db, "CREATE INDEX prices_rarity_index ON prices(rarity)")
	exec(db, "CREATE INDEX prices_low_index ON prices(price_low)")
	exec(db, "CREATE INDEX prices_median_index ON prices(price_median)")
	exec(db, "CREATE INDEX prices_high_index ON prices(price_high)")
	exec(db, "CREATE INDEX prices_names_trgm ON prices USING GIN(name gin_trgm_ops)")

	exec(db, "CREATE INDEX sets_type_index ON sets(type)")
	exec(db, "CREATE INDEX sets_border_index ON sets(border)")

	return nil
}
