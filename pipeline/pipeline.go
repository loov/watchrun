package pipeline

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/loov/hrtime"
	"github.com/loov/watchrun/pgroup"
)

type Log interface {
	Info(args ...any)
	Infof(format string, args ...any)

	Error(args ...any)
	Errorf(format string, args ...any)
}

type Process struct {
	Cmd  string
	Args []string
}

func (proc *Process) String() string {
	return proc.Cmd + " " + strings.Join(proc.Args, " ")
}

type Pipeline struct {
	Dir       string
	Output    io.Writer
	Log       Log
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

	output := pipe.Output
	if output == nil {
		output = os.Stdout
	}

	go io.Copy(output, pipe.reader)

	for _, proc := range pipe.Processes {
		pipe.mu.Lock()
		if pipe.killed {
			pipe.mu.Unlock()
			return
		}

		pipe.proc = proc
		pipe.active = exec.Command(proc.Cmd, proc.Args...)
		pipe.active.Dir = pipe.Dir
		pgroup.Setup(pipe.active)

		pipe.active.Stdout, pipe.active.Stderr = pipe.writer, pipe.writer

		pipe.Log.Info("<<  run:", proc.String(), ">>")

		start := hrtime.Now()
		err := pipe.active.Start()
		if err != nil {
			pipe.active = nil
			pipe.killed = true
			pipe.closeio()
			pipe.mu.Unlock()
			pipe.Log.Error("<< fail:", err, ">>")
			return
		}
		cmd := pipe.active
		pipe.mu.Unlock()

		if err := cmd.Wait(); err != nil {
			return
		}
		pipe.Log.Info("<< done:", proc.String(), hrtime.Since(start), ">>")
	}
}

func (pipe *Pipeline) Kill() {
	pipe.mu.Lock()
	defer pipe.mu.Unlock()

	if pipe.active != nil {
		pipe.Log.Info("<< kill:", pipe.proc.String(), ">>")
		pipe.closeio()
		pgroup.Kill(pipe.active)
		pipe.active = nil
	}
	pipe.killed = true
}

func Run(log Log, procs []Process) *Pipeline {
	pipe := &Pipeline{Log: log, Processes: procs}
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
