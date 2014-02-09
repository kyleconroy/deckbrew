CREATE EXTENSION pg_trgm;

CREATE TABLE cards (
    id                varchar(32)    primary key,
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
    record            text           DEFAULT '',
    rules             text           DEFAULT '',
    loyalty           integer        DEFAULT 0,
    cmc               integer        DEFAULT 0
);

CREATE INDEX cards_name_id ON cards(id);
CREATE INDEX cards_power_index ON cards(power);
CREATE INDEX cards_toughness_index ON cards(toughness);
CREATE INDEX cards_names_sort_index ON cards(name);

CREATE INDEX cards_types_index ON cards USING GIN(types);
CREATE INDEX cards_subtypes_index ON cards USING GIN(subtypes);
CREATE INDEX cards_supertypes_index ON cards USING GIN(supertypes);
CREATE INDEX cards_colors_index ON cards USING GIN(colors);
CREATE INDEX cards_sets_index ON cards USING GIN(sets);
CREATE INDEX cards_rarities_index ON cards USING GIN(rarities);
CREATE INDEX cards_status_index ON cards USING GIN(status);
CREATE INDEX cards_formats_index ON cards USING GIN(formats);
CREATE INDEX cards_mids_index ON cards USING GIN(mids);

CREATE INDEX cards_names_index ON cards USING GIN(name gin_trgm_ops);
CREATE INDEX cards_rules_index ON cards USING GIN(rules gin_trgm_ops);

CREATE TABLE sets (
    id                varchar(3) primary key,
    name              varchar(200) DEFAULT '',
    border            varchar(40) DEFAULT '',
    type              varchar(32) DEFAULT ''
);

CREATE INDEX sets_type_index ON sets(type);
CREATE INDEX sets_border_index ON sets(border);

GRANT ALL PRIVILEGES ON TABLE cards TO urza;
GRANT ALL PRIVILEGES ON TABLE sets TO urza;
