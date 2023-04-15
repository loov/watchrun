`watchrun` is a command-line utility to monitor a directory and run a command
when any file in that directory changes. It excludes temporary and hidden files
that start with `.` or `~` or end with `~`.

To install from source:

```
go get github.com/loov/watchrun
```

Example usage:

```
$ watchrun "go build -o example.exe . == ./example.exe"
```

_Note: directly using `watchrun go run main.go`, doesn't kill the compiled program automatically, which may cause problems, if you have a server listening._

Then you can test with:

```
$ echo package main; main(){ println("hello") } > main.go
$ echo package main; main(){ println("world") } > main.go
```

You can explicitly specify which folder or file to watch with `-monitor`:

```
$ watchrun -monitor ../../  "go build -o example.exe . == ./example.exe"
$ watchrun -monitor main.go "go build -o example.exe . == ./example.exe"
```

You can run multiple commands in succession with `==` (instead of the usual `&&`). For example:

```
$ watchrun go build -i . == myproject
```

## Usage

```
Usage of watchrun:
  -ignore string
        ignore files/folders that match these globs (default "~*;.*;*~;*.exe")
  -interval duration
        interval to wait between monitoring (default 300ms)
  -monitor string
        files/folders/globs to monitor (default ".")
  -recurse
        when watching a folder should recurse (default true)
  -verbose
        verbose output
```