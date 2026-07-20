`watchrun` is a command-line utility to monitor a directory and run a command
when any file in that directory changes. It excludes temporary and hidden files
that start with `.` or `~` or end with `~`.

To install from source:

```
go install github.com/loov/watchrun@latest
```

Example usage:

```
$ watchrun "go build -o example.exe . == ./example.exe"
```

_Note: directly using `watchrun go run main.go`, doesn't kill the compiled program automatically, which may cause problems, if you have a server listening._

Then you can test with:

```
$ echo 'package main; func main() { println("hello") }' > main.go
$ echo 'package main; func main() { println("world") }' > main.go
```

You can explicitly specify which folder or file to watch with `-monitor`:

```
$ watchrun -monitor ../../  "go build -o example.exe . == ./example.exe"
$ watchrun -monitor main.go "go build -o example.exe . == ./example.exe"
```

You can run multiple commands in succession with `==` or `;;` (instead of the usual `&&`). For example:

```
$ watchrun "go build . == ./myproject"
```

## Usage

```
Usage of watchrun:
  -care value
        check only changes to files that match these globs
  -clear
        clear the screen after rerunning the commands
  -ignore value
        ignore files/folders that match these globs (default .*;~*;*~;*.[ao];*.so;*.obj;*.log;*.test;*.prof;*.exe;*.dll)
  -interval duration
        interval to wait between monitoring (default 300ms)
  -log value
        logging level (debug, info, warn, error, silent)
  -monitor string
        files/folders/globs to monitor (default ".")
  -recurse
        when watching a folder should recurse (default true)
  -verbose
        verbose output (same as -log=debug)
```