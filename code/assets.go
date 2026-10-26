// Package school embeds the templates, static files and migrations so the
// compiled server is a single self-contained executable.
package school

import "embed"

//go:embed templates static migrations
var Assets embed.FS
