// Package web embeds the built React frontend.
package web

import "embed"

// FS holds web/dist; run `make web` to populate it.
//
//go:embed all:dist
var FS embed.FS
