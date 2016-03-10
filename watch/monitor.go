package watch

import (
	"os"
	"path/filepath"
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

	interval    time.Duration
	directories []string
	ignore      Filter
}

func New(path string, interval time.Duration, ignore ...Filter) *Watch {
	watch := &Watch{}
	watch.Changes = make(chan []Change)
	watch.interval = interval
	watch.directories = []string{path}
	watch.ignore = IgnoreAll(ignore...)
	watch.Start()
	return watch
}

func (watch *Watch) Stop() {
	watch.once.Do(func() {
		atomic.StoreInt32(&watch.stage, stopping)
		close(watch.Changes)
	})
}

func Changes(path string, interval time.Duration, ignore ...Filter) chan []Change {
	watch := New(path, interval, ignore...)
	return watch.Changes
}

func (watch *Watch) Wait() bool {
	<-watch.Changes
	return true
}

func (watch *Watch) Start() { go watch.Run() }

func (watch *Watch) Run() {
	defer func() { recover() }()

	defer atomic.StoreInt32(&watch.stage, stopped)

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
	for _, dir := range watch.directories {
		times.Merge(getFileTimes(dir, watch.ignore))
	}
	return times
}

type filetimes map[string]time.Time

func (into filetimes) Merge(other filetimes) {
	for name, time := range other {
		into[name] = time
	}
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

		abs := path
		if !filepath.IsAbs(abs) {
			full := filepath.Join(dir, path)
			abs, err = filepath.Abs(full)
			if err != nil {
				abs = full
			}
		}

		if ignore(abs, info) {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		if !info.IsDir() {
			times[abs] = info.ModTime()
		}
		return nil
	})
	return times
}
