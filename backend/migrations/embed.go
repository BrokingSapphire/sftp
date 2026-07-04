// Package migrations embeds the SQL migration files so they ship inside
// the compiled binary and run at startup.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
