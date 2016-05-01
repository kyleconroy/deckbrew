ci: deckbrew
	./deckbrew migrate
	./deckbrew sync
	go test -v ./...

deckbrew:
	go build
