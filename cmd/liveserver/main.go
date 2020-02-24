package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/loov/watchrun/jsreload"
)

var (
	addr     = flag.String("listen", ":9000", "port to listen to")
	dir      = flag.String("dir", ".", "directory to monitor")
	interval = flag.Duration("i", 300*time.Millisecond, "poll interval")
)

func DisableCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
}

func main() {
	flag.Parse()

	if !filepath.IsAbs(*dir) {
		absdir, err := filepath.Abs(*dir)
		if err == nil {
			*dir = absdir
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		DisableCache(w)
		path := filepath.FromSlash(path.Join(*dir, r.URL.Path))
		http.ServeFile(w, r, path)
	})
	http.Handle("/~reload.js", &jsreload.Server{
		Dir:      *dir,
		Interval: *interval,
	})

	fmt.Println("Server starting on:", *addr)
	fmt.Println("Watching folder:", *dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
