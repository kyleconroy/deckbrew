.PHONY: deps server test syncdb serverdb ami

brewapi: *.go
	godep go build -o brewapi

serve: brewapi
	./brewapi serve

test:
	godep go test -v

ami:
	packer build templates/api.json

prices.json: brewapi
	./brewapi price

imageami:
	packer build templates/image.json

clean:
	rm -f brewapi
	rm -rf deckbrew
