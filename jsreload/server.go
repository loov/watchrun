package jsreload

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loov/watchrun/watch"
)

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

	// URL where the jsreload server is serving on.
	// Code defaults to using the request.URL otherwise.
	URL string
	// ManualScriptSetup allows to disable automatic setup of js reloading script.
	ManualScriptSetup bool
	// ReconnectInterval defines how fast jsreload tries to reconnect after losing connection to the server.
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

// DefaultIgnore contains a list of files that you usually want to ignore.
// Such as temporary files, hidden files, log files and binaries.
var DefaultIgnore = watch.DefaultIgnore

// Server responds to regular requests with jsreload.Script and handles incoming websockets.
type Server struct {
	config    Config
	listeners *Hub
	watch     *watch.Watch
}

// NewServer creates a new server using the specified config.
func NewServer(config Config) *Server {
	if config.OnChange == nil {
		config.OnChange = func(change watch.Change) (string, Action) {
			return filepath.ToSlash(change.Path), ReloadBrowser
		}
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
//   GET with jsreload.Script.
//   WebSocket Upgrade with serving update messages.
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

	disableCache(w)

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

// disableCache ensures that client always re-reqiuests the file.
func disableCache(w http.ResponseWriter) {
	w.Header().Set("Expires", time.Unix(0, 0).Format(time.RFC1123))
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
}
