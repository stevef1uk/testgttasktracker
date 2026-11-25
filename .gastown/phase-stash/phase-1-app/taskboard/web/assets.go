package web

import "embed"

// FS embeds the HTML templates and static assets served by the TaskBoard web app.
//
//go:embed templates/*
var FS embed.FS
