CREATE TABLE cards (
    id                varchar(32) primary key,
    name              varchar(200) DEFAULT '',
    mana_cost         varchar(40) DEFAULT '',
    toughness         varchar(6) DEFAULT '',
    power             varchar(6) DEFAULT '',
    types             varchar(20)[] DEFAULT ARRAY[]::varchar[],
    subtypes          varchar(40)[] DEFAULT ARRAY[]::varchar[],
    supertypes        varchar(40)[] DEFAULT ARRAY[]::varchar[], 
    colors            varchar(40)[] DEFAULT ARRAY[]::varchar[],
    rules             text DEFAULT '',
    loyalty           smallint DEFAULT 0,
    cmc               smallint DEFAULT 0
);

CREATE INDEX cards_name_index ON cards(name);

CREATE TABLE sets (
    id                varchar(3) primary key,
    name              varchar(200) DEFAULT '',
    border            varchar(40) DEFAULT '',
    type              varchar(32) DEFAULT ''
);

GRANT ALL PRIVILEGES ON TABLE cards TO urza;
GRANT ALL PRIVILEGES ON TABLE sets TO urza;

