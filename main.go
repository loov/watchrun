package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/loov/hrtime"
	"github.com/loov/watchrun/pgroup"
	"github.com/loov/watchrun/watch"
)

var (
	ignore   = watch.Globs{NoDefault: false, Default: watch.DefaultIgnore, Additional: nil}
	care     = watch.Globs{NoDefault: false, Default: nil, Additional: nil}
	loglevel = LogLevelInfo

	interval = flag.Duration("interval", 300*time.Millisecond, "interval to wait between monitoring")
	monitor  = flag.String("monitor", ".", "files/folders/globs to monitor")
	recurse  = flag.Bool("recurse", true, "when watching a folder should recurse")
	verbose  = flag.Bool("verbose", false, "verbose output (same as -log=debug)")
)

func init() {
	flag.Var(&ignore, "ignore", "ignore files/folders that match these globs")
	flag.Var(&care, "care", "check only changes to files that match these globs")
	flag.Var(&loglevel, "log", "logging level (debug, info, warn, error, silent)")
}

func logln(at LogLevel, values ...any) {
	if loglevel.Matches(at) {
		fmt.Println(values...)
	}
}

func logf(at LogLevel, format string, values ...any) {
	if loglevel.Matches(at) {
		fmt.Printf(format, values...)
	}
}

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
	reader io.ReadCloser
	writer io.WriteCloser
	active *exec.Cmd
	killed bool
}

func (pipe *Pipeline) closeio() {
	pipe.reader.Close()
	pipe.writer.Close()
}

func (pipe *Pipeline) Run() {
	pipe.reader, pipe.writer = io.Pipe()
	go io.Copy(os.Stdout, pipe.reader)

	for _, proc := range pipe.Processes {
		pipe.mu.Lock()
		if pipe.killed {
			pipe.mu.Unlock()
			return
		}

		pipe.proc = proc
		pipe.active = exec.Command(proc.Cmd, proc.Args...)
		pgroup.Setup(pipe.active)

		pipe.active.Stdout, pipe.active.Stderr = pipe.writer, pipe.writer

		logln(LogLevelInfo, "<<  run:", proc.String(), ">>")

		start := hrtime.Now()
		err := pipe.active.Start()
		if err != nil {
			pipe.active = nil
			pipe.killed = true
			pipe.closeio()
			pipe.mu.Unlock()
			logln(LogLevelError, "<< fail:", err, ">>")
			return
		}
		cmd := pipe.active
		pipe.mu.Unlock()

		if err := cmd.Wait(); err != nil {
			return
		}
		logln(LogLevelInfo, "<< done:", proc.String(), hrtime.Since(start), ">>")
	}
}

func (pipe *Pipeline) Kill() {
	pipe.mu.Lock()
	defer pipe.mu.Unlock()

	if pipe.active != nil {
		logln(LogLevelInfo, "<< kill:", pipe.proc.String(), ">>")
		pipe.closeio()
		pgroup.Kill(pipe.active)
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
		if arg == ";;" || arg == "==" {
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

	if *verbose {
		loglevel = LogLevelDebug
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.PrintDefaults()
		return
	}
	procs := ParseArgs(args)

	monitoring := strings.Split(*monitor, ";")
	ignoring := ignore.All()
	caring := care.All()

	if loglevel.Matches(LogLevelDebug) {
		fmt.Println("Options:")
		fmt.Println("    interval   : ", *interval)
		fmt.Println("    recursive  : ", *recurse)
		fmt.Println("    monitoring : ", monitoring)
		fmt.Println("    ignoring   : ", ignoring)
		fmt.Println("    caring     : ", caring)
		fmt.Println()

		fmt.Println("Processes:")
		for _, proc := range procs {
			fmt.Printf("    %s %s\n", proc.Cmd, strings.Join(proc.Args, " "))
		}
		fmt.Println()
	}

	watcher := watch.New(
		*interval,
		monitoring,
		ignoring,
		caring,
		*recurse,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		watcher.Stop()
	}()

	var pipe *Pipeline
	for range watcher.Changes {
		if pipe != nil {
			pipe.Kill()
		}
		logln(LogLevelInfo, "<<", time.Now(), ">>")
		pipe = Run(procs)
	}

	if pipe != nil {
		pipe.Kill()
	}
}
