package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

type Process struct {
	Cmd  string
	Args []string
}

func (proc *Process) String() string {
	return proc.Cmd + " " + strings.Join(proc.Args, " ")
}

type Pipeline struct {
	Processes []Process

	mu     sync.Mutex
	proc   Process
	active *exec.Cmd
	killed bool
}

func (pipe *Pipeline) Run() {
	for _, proc := range pipe.Processes {
		pipe.mu.Lock()
		if pipe.killed {
			pipe.mu.Unlock()
			return
		}

		pipe.proc = proc
		pipe.active = exec.Command(proc.Cmd, proc.Args...)
		pipe.active.Stdout, pipe.active.Stderr = os.Stdout, os.Stdout

		fmt.Println("<<  run:", proc.String(), ">>")

		start := time.Now()
		err := pipe.active.Start()
		if err != nil {
			pipe.active = nil
			pipe.killed = true
			pipe.mu.Unlock()
			fmt.Println("<< fail:", err, ">>")
			return
		}
		cmd := pipe.active
		pipe.mu.Unlock()

		if err := cmd.Wait(); err != nil {
			return
		}
		fmt.Println("<< done:", proc.String(), time.Since(start), ">>")
	}
}

func (pipe *Pipeline) Kill() {
	pipe.mu.Lock()
	defer pipe.mu.Unlock()

	if pipe.active != nil {
		fmt.Println("<< kill:", pipe.proc.String(), ">>")
		pipe.active.Process.Kill()
		pipe.active = nil
	}
	pipe.killed = true
}

func Run(procs []Process) *Pipeline {
	pipe := &Pipeline{Processes: procs}
	go pipe.Run()
	return pipe
}

var ignoreexts []string

func ParseArgs(args []string) (procs []Process) {
	start := 0
	for i, arg := range args {
		if arg == ";;" {
			procs = append(procs, Process{
				Cmd:  args[start],
				Args: args[start+1 : i],
			})
			start = i + 1
		}
	}
	if start < len(args) {
		procs = append(procs, Process{
			Cmd:  args[start],
			Args: args[start+1:],
		})
	}

	return procs
}

func main() {
	flag.Parse()

	ignoreexts = strings.Split(*ignoreext, ";")

	args := flag.Args()
	if len(args) == 0 {
		flag.PrintDefaults()
		return
	}
	procs := ParseArgs(args)

	fmt.Printf("%#v\n", procs)

	pipe := Run(procs)
	for range monitor(*dir, *interval) {
		if pipe != nil {
			pipe.Kill()
		}
		fmt.Println("<<", time.Now(), ">>")
		pipe = Run(procs)
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
