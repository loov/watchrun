package main

import (
	"fmt"
	"strings"
)

type LogLevel int

const (
	LogLevelDebug  LogLevel = -4
	LogLevelInfo   LogLevel = 0
	LogLevelWarn   LogLevel = 4
	LogLevelError  LogLevel = 8
	LogLevelSilent LogLevel = 12
)

var logLevelName = map[LogLevel]string{
	LogLevelDebug:  "debug",
	LogLevelInfo:   "info",
	LogLevelWarn:   "warn",
	LogLevelError:  "error",
	LogLevelSilent: "silent",
}

func (level LogLevel) Matches(target LogLevel) bool {
	return target >= level
}

func (level *LogLevel) Set(name string) error {
	name = strings.ToLower(name)
	for l, n := range logLevelName {
		if n == name {
			*level = l
			return nil
		}
	}
	return fmt.Errorf("unknown log level %q, defaulting to debug\n", name)
}

func (level LogLevel) String() string {
	name, ok := logLevelName[level]
	if !ok {
		return fmt.Sprintf("LogLevel(%d)", level)
	}
	return name
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

type pipelineLog struct{}

func (pipelineLog) Info(args ...any) {
	logln(LogLevelInfo, args...)
}
func (pipelineLog) Infof(format string, args ...any) {
	logf(LogLevelInfo, format, args...)
}
func (pipelineLog) Error(args ...any) {
	logln(LogLevelError, args...)
}
func (pipelineLog) Errorf(format string, args ...any) {
	logf(LogLevelError, format, args...)
}
