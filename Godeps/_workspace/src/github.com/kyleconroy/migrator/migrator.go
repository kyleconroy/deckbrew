package migrator

import (
	"database/sql"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"time"
)

var create = `
CREATE TABLE IF NOT EXISTS migrations (
  filename varchar(300)  primary key,
  ran      timestamp     DEFAULT NOW()
)
`

var find = `
SELECT TRUE, ran
  FROM migrations
  WHERE filename=$1
`

var insert = `
INSERT INTO migrations (filename)
  VALUES ($1)
`

// Return a sorted list of SQL filenames
func migrations(path string) ([]string, error) {
	matches, err := filepath.Glob(path + "/*.sql")
	sort.Strings(matches)
	return matches, err
}

func logline(action, filename string) {
	log.Printf("action=%s filename=%s", action, filename)
}

func Run(db *sql.DB, path string) error {
	// Load the migrations file names into a slice
	// sort those files names
	files, err := migrations(path)
	if err != nil {
		return err
	}

	// Create the migrations table if it doesn't exist
	if _, err := db.Exec(create); err != nil {
		return err
	}

	for _, file := range files {
		name := filepath.Base(file)

		var found bool
		var timestamp time.Time
		err := db.QueryRow(find, name).Scan(&found, &timestamp)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if found {
			logline("skipped", name)
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		query, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		_, err = tx.Exec(string(query))
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(insert, name)
		if err != nil {
			tx.Rollback()
			return err
		}
		err = tx.Commit()
		if err != nil {
			return err
		}
		logline("created", name)
	}
	return nil
}
