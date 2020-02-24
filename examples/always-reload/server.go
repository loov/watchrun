package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/loov/watchrun/jsreload"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()

	http.Handle("/~reload.js", jsreload.NewServer(jsreload.Config{
		Monitor: []string{
			filepath.Join("static", "**"),
			filepath.Join("site", "**"),
		},
		Ignore: jsreload.DefaultIgnore,
	}))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", serveIndex)

	log.Println("listening on", *listen)
	http.ListenAndServe(*listen, nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("site/index.html"))
}
