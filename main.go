package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	dir      = flag.String("d", ".", "directory to watch")
	interval = flag.Duration("i", 300*time.Millisecond, "interval to wait between monitoring")
	verbose  = flag.Bool("v", false, "debug output")

	ignoreprefix = flag.String("p", "~.", "ignore files that start with any of the characters")
	ignoresuffix = flag.String("s", "~", "ignore files that end with any of the characters")
	ignoreext    = flag.String("e", ".exe", "ignore files that end with these suffixes")
)

func run(args []string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	return cmd
}

var ignoreexts []string

func main() {
	flag.Parse()

	ignoreexts = strings.Split(*ignoreext, ";")

	args := flag.Args()
	if len(args) == 0 {
		flag.PrintDefaults()
		return
	}

	cmd := run(args)
	for range monitor(*dir, *interval) {
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
		fmt.Println("<<", time.Now(), ">>")
		cmd = run(args)
	}
}

func monitor(path string, interval time.Duration) chan struct{} {
	ch := make(chan struct{})
	go func() {
		prev := make(filetimes)
		for {
			next := getfiletimes(path)
			if !prev.Same(next) {
				if *verbose {
					fmt.Println("<< file changed >>")
				}
				time.Sleep(interval)
				next = getfiletimes(path)
				ch <- struct{}{}

				prev = next
				continue
			}
			time.Sleep(interval)
		}
	}()
	return ch
}

type filetimes map[string]time.Time

func (x filetimes) Same(y filetimes) bool {
	if len(x) != len(y) {
		return false
	}
	for file, time := range x {
		if !y[file].Equal(time) {
			return false
		}
	}
	return true
}

func first(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func last(s string) (r rune) {
	for _, x := range s {
		r = x
	}
	return r
}

func ignorefile(name string) bool {
	if strings.IndexRune(*ignoreprefix, first(name)) >= 0 {
		return true
	} else if strings.IndexRune(*ignoresuffix, last(name)) >= 0 {
		return true
	}

	ext := filepath.Ext(name)
	for _, x := range ignoreexts {
		if x == ext {
			return true
		}
	}

	return false
}

func getfiletimes(path string) filetimes {
	times := make(filetimes)
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := filepath.Base(path)
		if name == "." || name == ".." || name == "" {
			return nil
		}

		if ignorefile(name) {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		if !info.IsDir() {
			times[path] = info.ModTime()
		}
		return nil
	})
	return times
}
