package gui

import "embed"

// DistFS contains the production build static assets for the React GUI.
//
//go:embed dist/*
var DistFS embed.FS
