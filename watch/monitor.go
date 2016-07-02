package watch

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Filter func(path string, info os.FileInfo) bool

const (
	running  = 0
	stopping = 1
	stopped  = 2
)

type Watch struct {
	Changes chan []Change

	once  sync.Once
	stage int32

	interval time.Duration

	monitor []string
	ignore  []string
	recurse bool
}

func New(interval time.Duration, monitor, ignore []string, recurse bool) *Watch {
	watch := &Watch{}
	watch.Changes = make(chan []Change)
	watch.interval = interval
	watch.monitor = monitor
	if len(watch.monitor) == 0 {
		watch.monitor = []string{"."}
	}
	watch.ignore = ignore
	watch.recurse = recurse
	watch.Start()
	return watch
}

func (watch *Watch) Stop() {
	watch.once.Do(func() {
		atomic.StoreInt32(&watch.stage, stopping)
	})
}

func Changes(interval time.Duration, monitor, ignore []string, recurse bool) chan []Change {
	watch := New(interval, monitor, ignore, recurse)
	return watch.Changes
}

func (watch *Watch) Wait() bool {
	<-watch.Changes
	return true
}

func (watch *Watch) Start() { go watch.Run() }

func (watch *Watch) Run() {
	defer atomic.StoreInt32(&watch.stage, stopped)
	defer close(watch.Changes)

	previous := make(filetimes)
	for {
		if atomic.LoadInt32(&watch.stage) >= stopping {
			break
		}

		next := watch.getState()
		if !previous.Same(next) {
			time.Sleep(watch.interval)
			next = watch.getState()
			changes := previous.Changes(next)
			previous = next
			watch.Changes <- changes
			continue
		}
		time.Sleep(watch.interval)
	}
}

func (watch *Watch) getState() filetimes {
	times := make(filetimes)
	for _, glob := range watch.monitor {
		times.IncludeGlob(glob, watch.ignore, watch.recurse)
	}
	return times
}

type filetimes map[string]time.Time

func isnav(name string) bool {
	return name == "." || name == ".."
}

func matchany(patterns []string, name string) bool {
	name = cname(name)
	for _, pattern := range patterns {
		if match, _ := filepath.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func (times filetimes) IncludeGlob(glob string, ignore []string, recurse bool) error {
	if glob == "" {
		return times.IncludeDir(".", ignore, recurse)
	}

	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	for _, abs := range matches {
		f, err := os.Lstat(abs)
		if err != nil {
			continue
		}

		if recurse && f.IsDir() {
			times.IncludeDir(abs, ignore, recurse)
		}
		if f.Mode().IsRegular() {
			times[cname(abs)] = f.ModTime()
		}
	}

	return nil
}

func (times filetimes) IncludeDir(dir string, ignore []string, recurse bool) error {
	matches, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range matches {
		base := f.Name()
		abs := filepath.Join(dir, base)
		if isnav(base) || base == "" || matchany(ignore, base) {
			continue
		}

		if recurse && f.IsDir() {
			times.IncludeDir(abs, ignore, recurse)
		}
		if f.Mode().IsRegular() {
			times[cname(abs)] = f.ModTime()
		}
	}

	return nil
}

type Change struct {
	Kind     string
	Path     string
	Modified time.Time
}

func (current filetimes) Changes(next filetimes) (changes []Change) {
	// modified and deleted files
	for file, time := range current {
		ntime, nok := next[file]
		if !nok {
			changes = append(changes, Change{"delete", file, time})
			continue
		}
		if !ntime.Equal(time) {
			changes = append(changes, Change{"modify", file, ntime})
			continue
		}
	}
	// added files
	for file, ntime := range next {
		if _, ok := current[file]; !ok {
			changes = append(changes, Change{"create", file, ntime})
			continue
		}
	}
	return
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

func cname(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(name)
	}
	return name
}
