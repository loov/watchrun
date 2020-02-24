package jsreload

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/loov/watchrun/watch"
	"golang.org/x/net/websocket"
)

type Server struct {
	Dir string

	Interval time.Duration
}

func DisableCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "" {
		websocket.Handler(server.changes).ServeHTTP(w, r)
		return
	}

	DisableCache(w)

	url := "ws://" + r.Host + r.RequestURI
	data := strings.Replace(Script, "{{.DEFAULT_HOST}}", url, -1)
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(data))
}

type Message struct {
	Type string  `json:"type"`
	Data Changes `json:"data"`
}

type Changes []Change
type Change struct {
	Kind     string    `json:"kind"`
	Path     string    `json:"path"`
	Package  string    `json:"package"`
	Depends  []string  `json:"depends"`
	Modified time.Time `json:"modified"`
}

var ActiveConnections int32

func (server *Server) changes(conn *websocket.Conn) {
	defer conn.Close()

	fmt.Println("CONNECTED", conn.LocalAddr())
	defer fmt.Println("DISCONNECTED", conn.LocalAddr())

	watcher := watch.New(server.Interval, nil, nil, nil, true)
	defer watcher.Stop()

	go func() {
		io.Copy(ioutil.Discard, conn)
		conn.Close()
		watcher.Stop()
	}()

	for changeset := range watcher.Changes {
		message := Message{
			Type: "changes",
		}
		for _, change := range changeset {
			rel, err := filepath.Rel(server.Dir, change.Path)
			if err != nil {
				rel = change.Path
			}
			path := filepath.ToSlash(rel)
			pkgname, depends := extractPackageInfo(change.Path)
			if pkgname == "" {
				pkgname = path
			}
			message.Data = append(message.Data, Change{
				Kind:     change.Kind,
				Path:     path,
				Modified: change.Modified,
				Package:  pkgname,
				Depends:  depends,
			})
		}

		if err := websocket.JSON.Send(conn, message); err != nil {
			return
		}
	}
}

var (
	rxPackage = regexp.MustCompile(`\bpackage\s*\(\s*"([^"]+)"\s*,\s*function`)
	rxDepends = regexp.MustCompile(`\bdepends\s*\(\s*"([^"]+)"\s*\)`)
)

func extractPackageInfo(filename string) (pkgname string, depends []string) {
	depends = []string{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}

	if m := rxPackage.FindStringSubmatch(string(data)); len(m) > 0 {
		pkgname = m[1]
	}
	for _, dependency := range rxDepends.FindAllStringSubmatch(string(data), -1) {
		depends = append(depends, dependency[1])
	}
	return
}
