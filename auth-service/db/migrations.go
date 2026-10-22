// Package migrations embeds the golang-migrate SQL files (db/migrations/*.sql).
package migrations

import "embed"

// FS holds the migration files under the "migrations" directory.
//
//go:embed migrations/*.sql
var FS embed.FS

// Dir is the directory inside FS that contains the migration files.
const Dir = "migrations"
