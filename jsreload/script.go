package jsreload

// Script is reloading script for server.
const Script = `
(function(global){
	"use strict";

	global.Reloader = Reloader;
	global.Reloader.defaultHost = "{{.DEFAULT_HOST}}";
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
