//go:build !windows
// +build !windows

package main

import "fmt"

func ClearScreen() {
	const clear = "\033[2J"
	const moveTopLeft = "\033[H"
	fmt.Print(clear + moveTopLeft)
}
