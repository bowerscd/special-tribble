package site

import "embed"

//go:embed index.html
//go:embed css/* scripts/*.js templates/*
var EmbeddedSite embed.FS
