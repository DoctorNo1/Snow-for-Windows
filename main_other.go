//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Snow for Windows runs on Windows only. Build it with: GOOS=windows go build -ldflags=-H=windowsgui")
	os.Exit(1)
}
