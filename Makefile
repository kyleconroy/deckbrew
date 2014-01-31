all:
	go run mtgjson.go database.go api.go

test: cards.json
	go test

syncdb: cards.json
	dropdb deckbrew
	psql -a -f schema/database.sql
	psql -d deckbrew -a -f schema/brew.sql
	go run api.go database.go mtgjson.go load cards.json

cards.json:
	wget http://mtgjson.com/json/AllSets-x.json.zip
	unzip AllSets-x.json.zip
	mv mnt/compendium/DevLab/mtgjson/web/json/AllSets-x.json cards.json
	rm -f AllSets-x.json.zip
	rm -rf mnt


