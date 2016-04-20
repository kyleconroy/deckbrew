package config

import (
	"fmt"
	"os"

	"stackmachine.com/cql"
)

type Config struct {
	DB   *cql.DB
	Port string

	HostImage string
	HostAPI   string
	HostWeb   string
}

func env(key, empty string) string {
	value := os.Getenv(key)
	if value == "" {
		return empty
	}
	return value
}

func FromEnv() (*Config, error) {

	// Configure the database
	url := env("DATABASE_URL", "")
	if url == "" {
		return nil, fmt.Errorf("connection requires DATABASE_URL environment variable")
	}

	db, err := cql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	port := env("PORT", "3000")
	return &Config{
		DB:        db,
		Port:      port,
		HostImage: env("DECKBREW_IMAGE_HOST", "deckbrew.image:"+port),
		HostAPI:   env("DECKBREW_API_HOST", "deckbrew.api:"+port),
		HostWeb:   env("DECKBREW_WEB_HOST", "deckbrew.web:"+port),
	}, nil
}
