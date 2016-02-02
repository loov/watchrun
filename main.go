package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/loov/watchrun/watch"
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

	args := flag.Args()
	if len(args) == 0 {
		flag.PrintDefaults()
		return
	}
	procs := ParseArgs(args)

	changes := watch.Changes(
		*dir, *interval,
		watch.IgnoreExtensions(strings.Split(*ignoreext, ";")...),
		watch.IgnoreNameSuffixed(strings.Split(*ignoresuffix, "")...),
		watch.IgnoreNamePrefixed(strings.Split(*ignoreprefix, "")...),
	)

	var pipe *Pipeline
	for range changes {
		if pipe != nil {
			pipe.Kill()
		}
		fmt.Println("<<", time.Now(), ">>")
		pipe = Run(procs)
	}
}
