package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html
var templates embed.FS

// FS holds the HTML templates served by the taskboard UI, rooted at the web directory.
var FS fs.FS = templates
