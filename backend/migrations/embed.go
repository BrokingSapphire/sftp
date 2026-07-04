// Package migrations embeds the goose SQL migrations so they ship inside the
// compiled binary and can be applied automatically at startup.
package migrations

import "embed"

//go:embed sftp/*.sql
var FS embed.FS
