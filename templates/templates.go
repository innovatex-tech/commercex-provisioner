package templates

import "embed"

// FS exports the deployment templates
//go:embed *.tmpl
var FS embed.FS
