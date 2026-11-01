package db

import "embed"

// Migrations contains the embedded Goose migration files.
//
//go:embed migrations/*.sql
var Migrations embed.FS
