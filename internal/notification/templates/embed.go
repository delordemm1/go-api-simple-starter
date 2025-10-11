package templates

import "embed"

// EmbeddedFS holds the embedded templates for production builds.
//go:embed files/*.tmpl
var EmbeddedFS embed.FS