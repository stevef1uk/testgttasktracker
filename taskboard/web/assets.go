package web

import "embed"

// Templates holds the HTML templates served by the taskboard UI.
//
//go:embed templates/*.html
var Templates embed.FS
