# Migrator <travis.yml link> <godoc link>

Migrator is a simple framework for running and maintaing database migrations in
Go. If you're looking for a full-featured migration framework, look elsewhere,
as Migrator is kept intentionally simple.

## Usage

```golang

package main

import (
    "github.com/kyleconroy/migrator"
)

func main() {
    err := migrator.Run(db, "migrations")
}
```

## How it Works

Migrator creates a `migrations` table in the database, in which it keeps track
of what migrations have been run. Migrations are just SQL files stored in a
directory. The migration order is determined by the putting all the SQL
filenames into a slice and using `sort.String`.

## Development
