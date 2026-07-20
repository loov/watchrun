package pipeline

import (
	"bytes"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type nopLog struct{}

func (nopLog) Info(args ...any)                  {}
func (nopLog) Infof(format string, args ...any)  {}
func (nopLog) Error(args ...any)                 {}
func (nopLog) Errorf(format string, args ...any) {}

func TestRunFlushesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no echo binary on windows")
	}
	var buf bytes.Buffer
	pipe := &Pipeline{
		Output:    &buf,
		Log:       nopLog{},
		Processes: []Process{{Cmd: "echo", Args: []string{"hello"}}},
	}
	pipe.Run()
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("output not flushed by the time Run returns: %q", buf.String())
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		exp  []Process
	}{
		{[]string{"echo", "hi"}, []Process{{"echo", []string{"hi"}}}},
		{[]string{"a", ";;", "b", "x"}, []Process{{"a", []string{}}, {"b", []string{"x"}}}},
		{[]string{";;", "b"}, []Process{{"b", []string{}}}},
		{[]string{"a", ";;", ";;", "b"}, []Process{{"a", []string{}}, {"b", []string{}}}},
		{[]string{"a", ";;"}, []Process{{"a", []string{}}}},
		// whole pipeline as a single quoted argument
		{[]string{"go build -o example.exe . == ./example.exe"},
			[]Process{{"go", []string{"build", "-o", "example.exe", "."}}, {"./example.exe", []string{}}}},
		{[]string{"a x ;; b"}, []Process{{"a", []string{"x"}}, {"b", []string{}}}},
		// single argument without separators stays a single command
		{[]string{"/path with spaces/cmd"}, []Process{{"/path with spaces/cmd", []string{}}}},
	}
	for _, test := range tests {
		got := ParseArgs(test.args)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("ParseArgs(%q) = %v, expected %v", test.args, got, test.exp)
		}
	}
}
