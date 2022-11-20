(function(exports){
	"use strict";

	exports.Client = Client;
	exports.Client.socketURL = "{{.SocketURL}}";

	function Client(host) {
		var client = this;

		var socket = new WebSocket(host || Client.socketURL);
		client.socket = socket;

		socket.addEventListener("open", function(ev) {
			socket.send({"type": "hello"});
			console.debug("watchjs open: ", ev);
		});

		socket.addEventListener("error", function(ev) {
			console.debug("watchjs error: ", ev);
		});

		socket.addEventListener("close", function(ev) {
			console.debug("watchjs close: ", ev);
			window.setInterval(function(){
				reconnect(host);
			}, {{.ReconnectInterval}});
		});

		socket.addEventListener("message", function(ev) {
			client.message(ev);
		})

		this.changeset = 0;
	}

	function reconnect(host) {
		try {
			var socket = new WebSocket(host || Client.socketURL);
			socket.addEventListener("open", function(){
				location.reload();
			});
		} catch(error) {
			console.debug("watchjs tried to reconnect and failed");
		}
	}

	Client.prototype = {
		message: function(ev) {
			var client = this;

			client.changeset++;
			if (client.changeset <= 1) {
				return;
			}

			var message = JSON.parse(ev.data);
			client["on" + message.type].call(client, message.data);
		},
		onhello: function(mesage){
			console.debug("watchjs server says hello");
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
						asset.src = name + "?" + Date.now();
						return asset;
					case ".css":
						var asset = document.createElement("link");
						asset.id = name;
						asset.href = name + "?" + Date.now();
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
						console.debug("don't know how to handle " + change.path + " reloading page");
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
				remove(change);
				inject(change);
			}

			for (var i = 0; i < changes.length; i++) {
				var change = changes[i];
				switch(change.action){
				case "ignore":
					continue;
				case "reload":
					location.reload();
					return;
				case "inject":
				}

				console.debug("live updating " + change.path);
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

	if({{.AutoSetup}}) {
		exports.instance = new Client();
	}
})(window.watchjs = {});
