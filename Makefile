.PHONY: deps server test syncdb serverdb ami

brewapi: *.go
	go build -o brewapi

deps:
	go get -d -v ./...

serve: brewapi prices.json
	./brewapi 

test: cards.json 
	go test -v

syncdb: brewapi cards.json 
	./brewapi load cards.json

prices.json:
	./brewapi price prices.json

ami: deckbrew
	packer build templates/api.json

imageami:
	packer build templates/image.json

deckbrew: Makefile *.go
	mkdir -p deckbrew
	cp *.go deckbrew
	cp -r formats deckbrew
	cp Makefile deckbrew

cards.json:
	wget http://mtgjson.com/json/AllSets-x.json.zip
	unzip AllSets-x.json.zip
	mv AllSets-x.json cards.json
	rm -f AllSets-x.json.zip

noami:
	@echo "DATABASE_PASSWORD and DATABASE_USER need to be set" && exit 1

clean:
	rm -f brewapi
	rm -rf deckbrew
