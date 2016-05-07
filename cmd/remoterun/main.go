package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/loov/watchrun/pgroup"
)

var (
	addr    = flag.String("addr", ":8080", "port to send/listen commands from")
	send    = flag.Bool("send", false, "try to send files to the addr")
	verbose = flag.Bool("verbose", false, "verbose output")
)

type File struct {
	Name string
	Data []byte
}

func sendfiles() {
	if len(flag.Args()) == 0 {
		log.Fatal(errors.New("Not enough files specified."))
		return
	}

	client, err := rpc.DialHTTP("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	for _, file := range flag.Args() {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		var reply int
		err = client.Call("Server.Start", &File{file, data}, &reply)
		if err != nil {
			log.Fatal(err)
		}
	}
}

type Server struct {
	mu     sync.Mutex
	active *Process
}

func (server *Server) Start(file *File, reply *int) (err error) {
	fmt.Println("<< received:", file.Name, ">>")

	err = ioutil.WriteFile(file.Name, file.Data, 0777)
	if err != nil {
		return err
	}

	server.mu.Lock()
	if server.active != nil {
		go server.active.Kill()
	}
	server.active = Run(file.Name, []string{})
	server.mu.Unlock()

	return nil
}

func main() {
	flag.Parse()
	if *send {
		sendfiles()
		return
	}

	server := &Server{}
	rpc.Register(server)
	rpc.HandleHTTP()
	fmt.Println("Server started on:", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

func Run(name string, args []string) *Process {
	proc := &Process{}
	proc.Name = name
	proc.Args = args
	go proc.Run()
	return proc
}

type Process struct {
	Name string
	Args []string

	mu     sync.Mutex
	reader io.ReadCloser
	writer io.WriteCloser
	active *exec.Cmd
	killed bool
}

func (proc *Process) closeio() {
	proc.reader.Close()
	proc.writer.Close()
}

func (proc *Process) Run() {
	proc.mu.Lock()

	proc.reader, proc.writer = io.Pipe()
	go io.Copy(os.Stdout, proc.reader)

	proc.active = exec.Command(proc.Name, proc.Args...)
	pgroup.Setup(proc.active)
	proc.active.Stdout, proc.active.Stderr = proc.writer, proc.writer

	fmt.Println("<<  run:", proc.Name, ">>")

	start := time.Now()
	err := proc.active.Start()
	if err != nil {
		proc.active = nil
		proc.killed = true
		proc.closeio()
		proc.mu.Unlock()
		fmt.Println("<< fail:", err, ">>")
		return
	}
	cmd := proc.active
	proc.mu.Unlock()

	if err := cmd.Wait(); err != nil {
		return
	}
	fmt.Println("<< done:", proc.Name, time.Since(start), ">>")
}

func (proc *Process) Kill() {
	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.active != nil {
		fmt.Println("<< kill:", proc.Name, ">>")
		proc.closeio()
		pgroup.Kill(proc.active)
		proc.active = nil
	}
	proc.killed = true
}
