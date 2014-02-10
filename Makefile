.PHONY: deps server test syncdb serverdb ami

brewapi: api.go mtgjson.go database.go qc.go etl.go search.go
	go build -o brewapi

deps:
	go get -d -v ./...

serve: brewapi
	./brewapi

test: cards.json 
	go test -v

syncdb: brewapi cards.json 
	./brewapi load cards.json

ami: deckbrew
ifndef DATABASE_PASSWORD
$(error The DATABASE_PASSWORD environment variable is not set)
endif

ifndef DATABASE_USER
$(error The DATABASE_USER environment variable is not set)
endif
	packer build template.json

deckbrew: Makefile *.go schema/*.sql
	mkdir -p deckbrew
	cp *.go deckbrew
	cp -r formats deckbrew
	cp Makefile deckbrew

cards.json:
	wget http://mtgjson.com/json/AllSets-x.json.zip
	unzip AllSets-x.json.zip
	mv mnt/compendium/DevLab/mtgjson/web/json/AllSets-x.json cards.json
	rm -f AllSets-x.json.zip
	rm -rf mnt

clean:
	rm -f brewapi
	rm -rf deckbrew
