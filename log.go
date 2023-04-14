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
