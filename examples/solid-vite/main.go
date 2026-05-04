package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/zackarysantana/sash"
	"github.com/zackarysantana/sash/examples/solid-vite/api"
	"github.com/zackarysantana/sash/examples/solid-vite/sashbindings"
	"github.com/zackarysantana/sash/src/sashsse"
)

//go:embed all:web/dist
var distRoot embed.FS

func main() {
	sub, err := fs.Sub(distRoot, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	broker := sashsse.NewBroker()
	svc := api.New(broker)

	opts := sash.Options{
		Title:       "Solid + Vite",
		Width:       520,
		Height:      420,
		OpenBrowser: true,
		DevURL:      os.Getenv("SASH_DEV_URL"),
		DevAPIAddr:  sashbindings.DevListenAddr,
		DevAPIMount: func(mux *http.ServeMux) {
			sashbindings.MountDevRoutes(mux, svc)
			mux.Handle("/events", sashsse.WithCORS(broker))
		},
		Assets: http.FS(sub),
		MountAPI: func(mux *http.ServeMux) {
			sashbindings.MountEmbedded(mux, svc)
			mux.Handle(sashbindings.EmbeddedMountPath+"/events", sashsse.WithCORS(broker))
		},
	}
	if err := sash.Run(opts); err != nil {
		log.Fatal(err)
	}
}
