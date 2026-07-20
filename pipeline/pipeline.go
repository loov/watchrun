package pipeline

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"slices"
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

	copied := make(chan struct{})
	go func() {
		defer close(copied)
		_, _ = io.Copy(output, pipe.reader)
	}()
	// close the writer when done, so io.Copy finishes and
	// all output has reached pipe.Output when Run returns
	defer func() {
		pipe.writer.Close()
		<-copied
	}()

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

// tokenize splits a command line like a POSIX shell: on whitespace,
// honoring single quotes, double quotes and backslash escapes.
// It does not expand variables or globs.
func tokenize(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	pending := false // cur holds a token, even if empty (e.g. '')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case ' ', '\t', '\n', '\r':
			if pending {
				tokens = append(tokens, cur.String())
				cur.Reset()
				pending = false
			}
		case '\'':
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, errors.New("unclosed single quote")
			}
			cur.WriteString(s[i+1 : i+1+end])
			i += end + 1
			pending = true
		case '"':
			i++
			closed := false
			for ; i < len(s); i++ {
				if s[i] == '"' {
					closed = true
					break
				}
				if s[i] == '\\' && i+1 < len(s) && strings.IndexByte(`"\$`+"`", s[i+1]) >= 0 {
					i++
				}
				cur.WriteByte(s[i])
			}
			if !closed {
				return nil, errors.New("unclosed double quote")
			}
			pending = true
		case '\\':
			if i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
			}
			pending = true
		default:
			cur.WriteByte(c)
			pending = true
		}
	}
	if pending {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

func ParseArgs(args []string) (procs []Process) {
	// support passing the whole pipeline as a single quoted argument,
	// since unquoted ";;" and "==" are mangled by shells
	if len(args) == 1 {
		fields, err := tokenize(args[0])
		if err != nil {
			fields = strings.Fields(args[0])
		}
		// ponytail: a quoted "==" still acts as a separator;
		// track quoting in tokenize if that ever matters
		if slices.Contains(fields, ";;") || slices.Contains(fields, "==") {
			args = fields
		}
	}

	start := 0
	for i, arg := range args {
		if arg == ";;" || arg == "==" {
			if i > start {
				procs = append(procs, Process{
					Cmd:  args[start],
					Args: args[start+1 : i],
				})
			}
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
