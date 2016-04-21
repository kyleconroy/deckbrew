package api

import (
	"github.com/kyleconroy/deckbrew/config"
	"github.com/kyleconroy/migrator"
	_ "github.com/lib/pq"
)

func MigrateDatabase() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	return migrator.Run(cfg.DB.DB, "migrations")
}
