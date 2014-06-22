.PHONY: deps server test syncdb serverdb ami

brewapi: *.go
	godep go build -o brewapi

serve: brewapi prices.json
	./brewapi 

test: cards.json 
	godep go test -v

ami: deckbrew
	packer build templates/api.json

imageami:
	packer build templates/image.json

deckbrew: Makefile *.go
	mkdir -p deckbrew
	cp *.go deckbrew
	cp -r formats deckbrew
	cp Makefile deckbrew

clean:
	rm -f brewapi
	rm -rf deckbrew
