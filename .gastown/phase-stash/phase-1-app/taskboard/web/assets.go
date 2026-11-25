package web

import "embed"

// FS embeds the HTML templates served by the TaskBoard web app.
// The static/ pattern is added once taskboard/web/static/ contains files
// (go:embed fails when a pattern matches nothing).
//
//go:embed templates/*
var FS embed.FS
