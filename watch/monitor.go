package watch

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Filter func(path string, info os.FileInfo) bool

type Monitor struct {
	Interval    time.Duration
	Directories []string
	Ignore      Filter
	Changes     chan struct{}
}

func Changes(path string, interval time.Duration, ignore ...Filter) chan struct{} {
	monitor := &Monitor{}
	monitor.Interval = interval
	monitor.Directories = []string{path}
	monitor.Changes = make(chan struct{})
	monitor.Ignore = IgnoreAll(ignore...)
	monitor.Start()

	return monitor.Changes
}

func (m *Monitor) Wait() bool {
	<-m.Changes
	return true
}

func (m *Monitor) Start() { go m.Run() }
func (m *Monitor) Run() {
	previous := make(filetimes)
	for {
		next := m.getState()
		if !previous.Same(next) {
			time.Sleep(m.Interval)
			previous = m.getState()
			m.Changes <- struct{}{}
			continue
		}
		time.Sleep(m.Interval)
	}
}

func (m *Monitor) getState() filetimes {
	times := make(filetimes)
	for _, dir := range m.Directories {
		times.Merge(getFileTimes(dir, m.Ignore))
	}
	return times
}

type filetimes map[string]time.Time

func (into filetimes) Merge(other filetimes) {
	for name, time := range other {
		into[name] = time
	}
}

func (a filetimes) Same(b filetimes) bool {
	if len(a) != len(b) {
		return false
	}
	for file, time := range a {
		if !b[file].Equal(time) {
			return false
		}
	}
	return true
}

func getFileTimes(dir string, ignore Filter) filetimes {
	times := make(filetimes)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore . and ..
		name := filepath.Base(path)
		if name == "." || name == ".." || name == "" {
			return nil
		}

		full := filepath.Join(dir, path)
		abs, err := filepath.Abs(full)
		if err != nil {
			abs = full
		}

		if ignore(abs, info) {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		if !info.IsDir() {
			times[cname(abs)] = info.ModTime()
		}
		return nil
	})
	return times
}

func cname(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(name)
	}
	return name
}
