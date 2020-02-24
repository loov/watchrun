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

// Config is configures server for modifications.
type Config struct {
	// Interval defines how often to poll the disk.
	Interval time.Duration
	// TODO: change to Monitor []string
	// Dir monitor this directory for changes.
	Dir string
	// Ignore these globs to avoid unnecessary updates.
	Ignore []string
	// Care only monitor files that match these globs.
	Care []string
}

// Reaction defines how browser reacts to a specific file changing.
type Reaction string

const (
	// IgnoreChanges ignores the file change.
	IgnoreChanges Reaction = "ignore"
	// ReloadBrowser reloads the whole page.
	ReloadBrowser Reaction = "reload"
	// LiveReload deletes old reference and reinjects the code.
	LiveReload Reaction = "live-reload"
)

// DefaultIgnore contains a list of files that you usually want to ignore.
// Such as temporary files, hidden files, log files and binaries.
var DefaultIgnore = watch.DefaultIgnore

// Server responds to regular requests with jsreload.Script and handles incoming websockets.
type Server struct {
	config Config
}

// NewServer creates a new server using the specified config.
func NewServer(config Config) *Server {
	return &Server{
		config: config,
	}
}

// disableCache ensures that client always re-reqiuests the file.
func disableCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
}

// ServeHTTP reponds to:
//   GET with jsreload.Script.
//   WebSocket Upgrade with serving update messages.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "" {
		websocket.Handler(server.changes).ServeHTTP(w, r)
		return
	}

	disableCache(w)

	url := "ws://" + r.Host + r.RequestURI
	data := strings.Replace(Script, "{{.DEFAULT_HOST}}", url, -1)
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(data))
}

// ReloadBrowser sends a reload message to all connected browsers.
func (server *Server) ReloadBrowser() {
	// TODO:
}

// Message is json message that is sent on changes.
type Message struct {
	Type string  `json:"type"`
	Data Changes `json:"data"`
}

// Changes is a list of Change.
type Changes []Change

// Change defines a list of changes.
type Change struct {
	// Kind is one of "create", "modify", "delete"
	Kind string `json:"kind"`
	// Path rewriting TODO:
	Path string `json:"path"`
	// Modified returns the modified time of the file.
	Modified time.Time `json:"modified"`

	// Package finds match based on `package("<pkgname>", function(){`
	Package string `json:"package"`
	// Depends finds text based on `depends("<pkgname>")`
	Depends []string `json:"depends"`
}

func (server *Server) changes(conn *websocket.Conn) {
	defer conn.Close()

	fmt.Println("CONNECTED", conn.LocalAddr())
	defer fmt.Println("DISCONNECTED", conn.LocalAddr())

	watcher := watch.New(server.config.Interval, []string{server.config.Dir}, server.config.Ignore, server.config.Care, true)
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
			rel, err := filepath.Rel(server.config.Dir, change.Path)
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
