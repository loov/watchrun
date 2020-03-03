package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/loov/watchrun/watch"
	"github.com/loov/watchrun/watchjs"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "address to listen to")
	flag.Parse()

	staticDir := filepath.Join("site", "static")

	var assets *Assets
	assets = &Assets{
		watchjs: watchjs.NewServer(watchjs.Config{
			Monitor: []string{
				filepath.Join("site", "**"),
			},
			Ignore: watchjs.DefaultIgnore,
			OnChange: func(change watch.Change) (string, watchjs.Action) {
				// When change is in staticDir, we instruct the browser live (re)inject the file.
				if url, ok := watchjs.FileToURL(change.Path, staticDir, "/static"); ok {
					if filepath.Ext(change.Path) == ".html" {
						assets.Recompile()
						return url, watchjs.IgnoreChanges
					}
					if filepath.Ext(change.Path) == ".css" {
						return url, watchjs.LiveInject
					}
					return url, watchjs.ReloadBrowser
				}
				return "/" + filepath.ToSlash(change.Path), watchjs.ReloadBrowser
			},
		}),
	}

	http.Handle("/~watch.js", assets.watchjs)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	server := &Server{
		Assets: assets,
	}

	http.HandleFunc("/", server.ServeIndex)
	http.HandleFunc("/other", server.ServeOther)

	log.Println("listening on", *listen)
	err := http.ListenAndServe(*listen, nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Server struct {
	Assets *Assets
}

func (server *Server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	t, err := server.Assets.T()
	if err != nil {
		errorPage.Execute(w, err)
		return
	}

	err = t.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

func (server *Server) ServeOther(w http.ResponseWriter, r *http.Request) {
	t, err := server.Assets.T()
	if err != nil {
		errorPage.Execute(w, err)
		return
	}

	err = t.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		log.Println(err)
		return
	}
}

type Assets struct {
	watchjs *watchjs.Server
	mu      sync.Mutex
	root    *template.Template
	err     error
}

func (assets *Assets) Recompile() {
	assets.mu.Lock()
	defer assets.mu.Unlock()

	assets.recompile()
	assets.watchjs.ReloadBrowser()
}

func (assets *Assets) recompile() {
	root, err := template.ParseGlob("site/**.html")
	assets.root = root
	assets.err = err
}

func (assets *Assets) T() (*template.Template, error) {
	assets.mu.Lock()
	defer assets.mu.Unlock()

	if assets.root == nil && assets.err == nil {
		assets.recompile()
	}

	return assets.root, assets.err
}

// errorPage is needed to ensure that after refresh "/~watch.js"
// gets lodaded. Otherwise it could end up breaking the refreshing.
var errorPage = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error</title>
    <script src="/~watch.js"></script>

    <script src="/static/main.js"></script>
    <link rel="stylesheet" href="/static/main.css">
</head>
<body>
    <h1>{{.}}</h1>
</body>
</html>
`))
