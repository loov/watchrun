package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/loov/watchrun/watchjs"
)

func DisableCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
}

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen")
	monitor := flag.String("monitor", ".", "directory to monitor changes")
	serve := flag.String("serve", ".", "directory to serve content")

	flag.Parse()

	if !filepath.IsAbs(*monitor) {
		abs, err := filepath.Abs(*monitor)
		if err == nil {
			*monitor = abs
		}
	}

	if !filepath.IsAbs(*serve) {
		abs, err := filepath.Abs(*serve)
		if err == nil {
			*serve = abs
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		DisableCache(w)
		path := filepath.FromSlash(path.Join(*serve, r.URL.Path))
		http.ServeFile(w, r, path)
	})

	http.Handle("/~watch.js", watchjs.NewServer(watchjs.Config{
		Monitor: []string{filepath.Join(*monitor, "**")},
		Ignore:  watchjs.DefaultIgnore,
	}))

	fmt.Println("Listening on:", *listen)
	fmt.Println("Monitoring:", *monitor)
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}
