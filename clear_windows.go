//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func ClearScreen() {
	stdout, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}

	var originalMode uint32
	if err := windows.GetConsoleMode(stdout, &originalMode); err != nil {
		return
	}
	if err := windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return
	}
	defer func() { _ = windows.SetConsoleMode(stdout, originalMode) }()

	const clearScreen = "\x1b[2J"
	const clearScrollBack = "\x1b[3J"
	const resetCursor = "\x1b[H"

	fmt.Fprint(os.Stdout, clearScreen+clearScrollBack+resetCursor)
}
