.PHONY: deps server test syncdb serverdb

DATABASE_URL := postgres://localhost/deckbrew?sslmode=disable

deckbrew: api.go mtgjson.go database.go
	go build -o deckbrew

deps:
	go get -d -v ./...


serve:
	go run mtgjson.go database.go api.go

test: cards.json
	go test

syncdb: cards.json
	-dropdb deckbrew
	-dropuser urza
	psql -a -f schema/database.sql
	psql -d deckbrew -a -f schema/brew.sql
	go run api.go database.go mtgjson.go load cards.json

cards.json:
	wget http://mtgjson.com/json/AllSets-x.json.zip
	unzip AllSets-x.json.zip
	mv mnt/compendium/DevLab/mtgjson/web/json/AllSets-x.json cards.json
	rm -f AllSets-x.json.zip
	rm -rf mnt


