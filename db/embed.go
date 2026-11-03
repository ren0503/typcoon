// Package db embeds the SQL migrations and schema shipped with the service.
package db

import "embed"

// Migrations contains the embedded Goose migration files.
//
//go:embed migrations/*.sql
var Migrations embed.FS
