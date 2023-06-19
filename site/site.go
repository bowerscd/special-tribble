package site

import (
	"embed"
	"net/http"
)

//go:embed index.html
//go:embed css/* scripts/*.js templates/*
var embeddedSite embed.FS

func WebRootHandler(server *http.ServeMux) {
	server.Handle("/", http.FileServer(http.FS(embeddedSite)))
}
