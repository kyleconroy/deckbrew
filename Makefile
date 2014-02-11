.PHONY: deps server test syncdb serverdb ami

ifdef $(DATABASE_HOST)
ifdef $(DATABASE_USER)
		ami := packami
endif
endif

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

packami: deckbrew
	packer build template.json


deckbrew: Makefile *.go
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

ami:
	@echo "DATABASE_HOST and DATABASE_USER need to be set" && exit 1

clean:
	rm -f brewapi
	rm -rf deckbrew
