// Package school embeds the templates, static files, migrations and test data
// so the compiled server is a single self-contained executable.
package school

import "embed"

//go:embed templates static migrations test.sql
var Assets embed.FS
