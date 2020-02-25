package watchjs_test

import (
	"path/filepath"
	"testing"

	"github.com/loov/watchrun/watchjs"
)

func TestFileToURL(t *testing.T) {
	type test struct {
		dir     string
		prefix  string
		relfile string

		expected string
	}

	fp := filepath.Join
	tests := []test{
		{fp("static", "css"), "static/", "main.css", "/static/main.css"},
		{fp("static", "css"), "static/", fp("alpha", "main.css"), "/static/alpha/main.css"},
		{fp(""), "static/", fp("alpha", "main.css"), "/static/alpha/main.css"},

		{fp("static", "css"), "/static/", "main.css", "/static/main.css"},
		{fp("static", "css"), "/static/", fp("alpha", "main.css"), "/static/alpha/main.css"},
		{fp(""), "/static/", fp("alpha", "main.css"), "/static/alpha/main.css"},

		{"", "", fp("static", "main.css"), "/static/main.css"},
	}

	for _, test := range tests {
		path, _ := watchjs.FileToURL(
			filepath.Join(test.dir, test.relfile),
			test.dir,
			test.prefix,
		)
		if path != test.expected {
			t.Errorf("got %q, expected %q", path, test.expected)
		}
	}
}
