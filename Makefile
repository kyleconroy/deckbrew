syncdb:
	dropdb deckbrew
	createdb deckbrew
	psql -d deckbrew -a -f schema/brew.sql
	go run api.go database.go
