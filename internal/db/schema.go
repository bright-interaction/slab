// Package db embeds the SQLite schema.
package db

import _ "embed"

//go:embed schema.sql
var Schema []byte
