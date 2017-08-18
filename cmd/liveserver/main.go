package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/loov/watchrun/watch"

	"golang.org/x/net/websocket"
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
	http.HandleFunc("/~reload.js", ServeReloader)

	fmt.Println("Server starting on:", *addr)
	fmt.Println("Watching folder:", *dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func ServeReloader(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "" {
		websocket.Handler(ServeChanges).ServeHTTP(w, r)
		return
	}

	DisableCache(w)

	url := "ws://" + r.Host + r.RequestURI
	data := strings.Replace(ReloaderJS, "DEFAULT_HOST", url, -1)
	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(data))
}

const ReloaderJS = `
(function(global){
	"use strict";

	global.Reloader = Reloader;
	global.Reloader.defaultHost = "DEFAULT_HOST";
	function Reloader(host) {
		this.socket = new WebSocket(host || Reloader.defaultHost);
		this.socket.onopen = function(ev) {
			console.log("reloader open: ", ev);
		};
		this.socket.onerror = function(ev) {
			console.log("reloader error: ", ev);
		};
		this.socket.onclose = function(ev) {
			console.log("reloader close: ", ev);
		};
		this.socket.onmessage = this.message.bind(this);
		this.changeset = 0;
	}

	Reloader.prototype = {
		message: function(ev) {
			var reloader = this;

			reloader.changeset++;
			if (reloader.changeset <= 1) {
				return;
			}

			var message = JSON.parse(ev.data);
			reloader["on" + message.type].call(reloader, message.data);
		},
		onchanges: function(changes) {
			var head = document.getElementsByTagName("head")[0];

			function pathext(name) {
				var i = name.lastIndexOf(".");
				if (i < 0) {
					return "";
				}
				return name.substring(i);
			}

			function makeasset(name) {
				switch (pathext(name)) {
					case ".js":
						var asset = document.createElement("script");
						asset.id = name;
						asset.src = name;
						return asset;
					case ".css":
						var asset = document.createElement("link");
						asset.id = name;
						asset.href = name;
						asset.rel = "stylesheet";
						return asset;
				}
				return undefined;
			}

			function findasset(name) {
				var el = document.getElementById(name);
				if (el) {
					return el;
				}

				switch (pathext(name)) {
					case ".js":
						return document.querySelector("script[src='" + name + "']");
					case ".css":
						return document.querySelector("script[href='" + name + "']");
				}
				return undefined;
			}

			function inject(change) {
				var el = findasset(change.path);
				if (el) {
					el.id = change.path;
				} else {
					var asset = makeasset(change.path);
					if (asset) {
						head.appendChild(asset);
					} else {
						location.reload();
					}
				}
			}

			function remove(change) {
				var el = findasset(change.path);
				if (el) {
					el.remove();
				}
			}

			function modify(change) {
				console.info("changed ", change.path)
				remove(change);
				inject(change);
			}

			for (var i = 0; i < changes.length; i++) {
				var change = changes[i];
				switch (change.kind) {
					case "create":
						inject(change);
						break;
					case "delete":
						remove(change);
						break;
					case "modify":
						modify(change);
						break;
				}
			}
		}
	};
})(window);
`

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

func ServeChanges(conn *websocket.Conn) {
	defer conn.Close()

	fmt.Println("CONNECTED", conn.LocalAddr())
	defer fmt.Println("DISCONNECTED", conn.LocalAddr())

	watcher := watch.New(*interval, nil, nil, nil, true)
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
			rel, err := filepath.Rel(*dir, change.Path)
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
