package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/loov/watchrun/jsreload"
	"github.com/loov/watchrun/watch"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()

	http.Handle("/~reload.js", jsreload.NewServer(jsreload.Config{
		Monitor: []string{
			filepath.Join("static", "**"),
			filepath.Join("templates", "**"),
		},
		Ignore: jsreload.DefaultIgnore,
		OnChange: func(change watch.Change) (string, jsreload.Reaction) {
			return "", jsreload.Ignore
		},
	}))

	http.HandleFunc("/", serveIndex)
	http.ListenAndServe(*listen, nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseGlob("templates/**")
	if err != nil {
		log.Println("failed to parse templates: %v", err)
		http.Error(w, fmt.Sprintf("failed to parse templates: %v", err), http.StatusInternalServerError)
		return
	}

	err := t.Execute(w, nil)
	if err != nil {
		log.Println("failed to execute template: %v", err)
	}
}
