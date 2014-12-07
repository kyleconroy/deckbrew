CREATE TABLE prices (
        multiverse_id     varchar(20)    NOT NULL,
        created           timestamp      DEFAULT now(),
        foil              boolean        DEFAULT false,
        low               integer        DEFAULT 0,
        median            integer        DEFAULT 0,
        high              integer        DEFAULT 0
);

CREATE INDEX prices_created_index ON prices(created);
CREATE INDEX prices_id_index ON prices(multiverse_id);
CREATE INDEX prices_low_index ON prices(low);
CREATE INDEX prices_median_index ON prices(median);
CREATE INDEX prices_high_index ON prices(high);
