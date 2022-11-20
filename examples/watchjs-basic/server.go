package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/loov/watchrun/watchjs"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()

	// This example assumes that your folder structure and URL structure match.
	// See "watchjs-live" example how to adjust for a different structure.
	http.Handle("/~watch.js", watchjs.NewServer(watchjs.Config{
		Monitor: []string{
			filepath.Join("static", "**"),
			filepath.Join("site", "**"),
		},
		Ignore: watchjs.DefaultIgnore,
	}))

	static := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		watchjs.DisableCache(w)
		static.ServeHTTP(w, r)
	})
	http.HandleFunc("/", serveIndex)

	log.Println("listening on", *listen)
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("site/index.html"))
}
