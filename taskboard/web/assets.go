package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.html static/*
var assets embed.FS

// FS holds the templates and static assets served by the taskboard UI, rooted at the web directory.
var FS fs.FS = assets
