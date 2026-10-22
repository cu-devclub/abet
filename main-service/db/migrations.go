// Package db embeds the golang-migrate SQL files so the binary can apply them at startup.
package db

import "embed"

// MigrationsDir is the directory inside FS that holds the NNNNNN_name.{up,down}.sql files.
const MigrationsDir = "migrations"

//go:embed migrations/*.sql
var FS embed.FS
