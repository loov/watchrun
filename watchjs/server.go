package watchjs

import (
	_ "embed"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loov/watchrun/watch"
)

// Script is reloading script for server.
//
//go:embed script.js
var Script string

// Config is configures server for modifications.
type Config struct {
	// Interval defines how often to poll the disk.
	Interval time.Duration
	// Monitor these globs for changes.
	Monitor []string
	// Ignore these globs to avoid unnecessary updates.
	Ignore []string
	// Care only monitor files that match these globs.
	Care []string

	// URL where the watchjs server is serving on.
	// Code defaults to using the request.URL otherwise.
	URL string
	// ManualScriptSetup allows to disable automatic setup of js reloading script.
	ManualScriptSetup bool
	// ReconnectInterval defines how fast watchjs tries to reconnect after losing connection to the server.
	ReconnectInterval time.Duration

	// OnChange should return the URL path for a particular file and the reaction for javascript.
	OnChange func(change watch.Change) (path string, reaction Action)
}

// Action defines how browser reacts to a specific file changing.
type Action string

const (
	// IgnoreChanges ignores the file change.
	IgnoreChanges Action = "ignore"
	// ReloadBrowser reloads the whole page.
	ReloadBrowser Action = "reload"
	// LiveInject deletes old reference and reinjects the code.
	LiveInject Action = "inject"
)

// FileToURL converts filename in basedir to a url in urlprefix.
func FileToURL(filename, basedir, urlprefix string) (url string, ok bool) {
	rel, err := filepath.Rel(basedir, filename)
	if err != nil {
		return "", false
	}

	return path.Join("/", urlprefix, filepath.ToSlash(rel)), true
}

// DefaultIgnore contains a list of files that you usually want to ignore.
// Such as temporary files, hidden files, log files and binaries.
var DefaultIgnore = watch.DefaultIgnore

// Server responds to regular requests with watchjs.Script and handles incoming websockets.
type Server struct {
	config    Config
	listeners *Hub
	watch     *watch.Watch
}

// NewServer creates a new server using the specified config.
func NewServer(config Config) *Server {
	if config.OnChange == nil {
		config.OnChange = DefaultOnChange
	}

	if config.ReconnectInterval == 0 {
		config.ReconnectInterval = time.Second
	}

	server := &Server{
		config:    config,
		listeners: NewHub(),
		watch: watch.New(
			config.Interval,
			config.Monitor,
			config.Ignore,
			config.Care,
			true,
		),
	}

	go server.monitor()

	return server
}

// DefaultOnChange assumes that your folder structure matches your URL structure.
// It live injects css and reloads browser otherwise.
func DefaultOnChange(change watch.Change) (string, Action) {
	url, ok := FileToURL(change.Path, "", "")
	if !ok {
		url = path.Join("/", filepath.ToSlash(change.Path))
	}
	if filepath.Ext(change.Path) == ".css" {
		return url, LiveInject
	}
	return url, ReloadBrowser
}

// monitor handles file changes and notifies connections.
func (server *Server) monitor() {
	for changeset := range server.watch.Changes {
		message := Message{
			Type: "changes",
		}

		for _, change := range changeset {
			path, action := server.config.OnChange(change)

			pkgname, depends := extractPackageInfo(change.Path)
			if pkgname == "" {
				pkgname = path
			}

			message.Data = append(message.Data, Change{
				Kind:     change.Kind,
				Path:     path,
				Modified: change.Modified,
				Action:   action,
				Package:  pkgname,
				Depends:  depends,
			})
		}

		server.listeners.Dispatch(message)
	}
}

// Stop stops changes monitoring.
func (server *Server) Stop() {
	server.watch.Stop()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  0,
	WriteBufferSize: 0,
}

// ServeHTTP reponds to:
//
//	GET with watchjs.Script.
//	WebSocket Upgrade with serving update messages.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "" {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		go server.changes(conn)
		return
	}

	DisableCache(w)

	url := server.config.URL
	if url == "" {
		url = "ws://" + r.Host + r.RequestURI
	}

	if trimmed := strings.TrimPrefix(url, "http://"); trimmed != url {
		url = "ws://" + trimmed
	} else if trimmed := strings.TrimPrefix(url, "https://"); trimmed != url {
		url = "wss://" + trimmed
	}

	data := Script
	data = strings.Replace(data, "{{.SocketURL}}", url, -1)

	autoSetup := "true"
	if server.config.ManualScriptSetup {
		autoSetup = "false"
	}
	data = strings.Replace(data, "{{.AutoSetup}}", autoSetup, -1)

	reconnectInterval := fmt.Sprintf("%d", server.config.ReconnectInterval.Milliseconds())
	data = strings.Replace(data, "{{.ReconnectInterval}}", reconnectInterval, -1)

	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(data))
}

// ReloadBrowser sends a reload message to all connected browsers.
func (server *Server) ReloadBrowser() {
	server.listeners.Dispatch(Message{
		Type: "changes",
		Data: []Change{
			{
				Path:   "*",
				Action: ReloadBrowser,
			},
		},
	})
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
	// Action is the action browser should take with this file.
	Action Action `json:"action"`

	// Package finds match based on `package("<pkgname>", function(){`
	Package string `json:"package"`
	// Depends finds text based on `depends("<pkgname>")`
	Depends []string `json:"depends"`
}
