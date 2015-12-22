`watchrun` is a command-line utility to monitor a directory and run a command
when any file in that directory changes. It excludes temporary and hidden files
that start with `.` or `~` or end with `~`.

To install from source:

```
go get github.com/loov/watchrun
```

Example usage:

```
$ watchrun go run main.go
```

Then you can test with:

```
$ echo package main; main(){ println("hello") } > main.go
$ echo package main; main(){ println("world") } > main.go
```

You can explicitly specify which folder to watch with `-d`:

```
$ watchrun -d ../../ go run main.go
```