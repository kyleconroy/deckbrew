CREATE TABLE cards (
    id                varchar(32) primary key,
    name              varchar(200),
    mana_cost         varchar(40),
    toughness         varchar(6),
    power             varchar(6),
    partner_card      varchar(32) references cards(id),
    types             varchar(20)[],
    subtypes          varchar(40)[],
    color_indicator   varchar(10)[],
    rules_text        text,
    loyalty           smallint default 0,
    converted_cost    smallint default 0
);
