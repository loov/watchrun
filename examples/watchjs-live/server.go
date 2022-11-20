package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/loov/watchrun/watch"
	"github.com/loov/watchrun/watchjs"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()

	staticDir := filepath.Join("site", "static")

	http.Handle("/~watch.js", watchjs.NewServer(watchjs.Config{
		Monitor: []string{
			filepath.Join("site", "**"),
		},
		Ignore: watchjs.DefaultIgnore,
		OnChange: func(change watch.Change) (string, watchjs.Action) {
			// When change is in staticDir, we instruct the browser live (re)inject the file.
			if url, ok := watchjs.FileToURL(change.Path, staticDir, "/static"); ok {
				if filepath.Ext(change.Path) == ".css" {
					return url, watchjs.LiveInject
				}
				return url, watchjs.ReloadBrowser
			}
			return "/" + filepath.ToSlash(change.Path), watchjs.ReloadBrowser
		},
	}))

	static := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
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
