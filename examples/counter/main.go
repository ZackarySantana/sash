package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/zackarysantana/sash"
	"github.com/zackarysantana/sash/examples/counter/api"
	"github.com/zackarysantana/sash/examples/counter/sashbindings"
)

//go:embed web/*
var webRoot embed.FS

func main() {
	sub, err := fs.Sub(webRoot, "web")
	if err != nil {
		log.Fatal(err)
	}

	svc := &api.API{}

	opts := sash.Options{
		Title:       "Counter",
		Width:       420,
		Height:      360,
		OpenBrowser: true,
		DevURL:      os.Getenv("SASH_DEV_URL"),
		DevAPIAddr:  sashbindings.DevListenAddr,
		DevAPIMount: func(mux *http.ServeMux) {
			sashbindings.MountDevRoutes(mux, svc)
		},
		Assets: http.FS(sub),
		MountAPI: func(mux *http.ServeMux) {
			sashbindings.MountEmbedded(mux, svc)
		},
	}
	if err := sash.Run(opts); err != nil {
		log.Fatal(err)
	}
}
