package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/loov/watchrun/pipeline"
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
	clear    = flag.Bool("clear", false, "clear the screen after rerunning the commands")
)

func init() {
	flag.Var(&ignore, "ignore", "ignore files/folders that match these globs")
	flag.Var(&care, "care", "check only changes to files that match these globs")
	flag.Var(&loglevel, "log", "logging level (debug, info, warn, error, silent)")
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
	procs := pipeline.ParseArgs(args)

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

	var pipe *pipeline.Pipeline
	for range watcher.Changes {
		if pipe != nil {
			pipe.Kill()
		}
		if *clear {
			ClearScreen()
		}
		logln(LogLevelInfo, "<<", time.Now(), ">>")
		pipe = pipeline.Run(pipelineLog{}, procs)
	}

	if pipe != nil {
		pipe.Kill()
	}
}
