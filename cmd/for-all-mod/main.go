package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/loov/watchrun/pipeline"
	"golang.org/x/sync/errgroup"
)

func main() {
	parallel := flag.Int("parallel", 4, "number of pipelines to run concurrently")

	flag.Parse()

	procs := pipeline.ParseArgs(flag.Args())

	modfiles := []string{}

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) == "go.mod" {
			modfiles = append(modfiles, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sort.Strings(modfiles)

	var group errgroup.Group
	group.SetLimit(*parallel)

	for _, modfile := range modfiles {
		group.Go(func() error {
			var output bytes.Buffer

			pipe := &pipeline.Pipeline{
				Dir:       filepath.Dir(modfile),
				Output:    &output,
				Log:       pipelineLog{},
				Processes: procs,
			}

			pipe.Run()

			fmt.Print(output.String())

			return nil
		})
	}
}

type pipelineLog struct{}

func (pipelineLog) Info(args ...any) {
}
func (pipelineLog) Infof(format string, args ...any) {
	fmt.Printf(format, args...)
}
func (pipelineLog) Error(args ...any) {
}
func (pipelineLog) Errorf(format string, args ...any) {
	fmt.Printf(format, args...)
}
