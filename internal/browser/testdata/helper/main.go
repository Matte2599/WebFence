package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "--grandchild" {
		time.Sleep(time.Second)
		_ = os.WriteFile(os.Args[2], []byte("survived"), 0600)
		return
	}
	if len(os.Args) != 2 || os.Args[1] != "--browser-helper" {
		os.Exit(2)
	}
	// The production helper follows this contract: no children or network
	// access are started until the parent has sent its bounded configuration.
	config, err := io.ReadAll(io.LimitReader(os.Stdin, 64<<10))
	if err != nil {
		os.Exit(2)
	}
	switch {
	case string(config) == "echo":
		fmt.Print("ok")
	case string(config) == "spam":
		fmt.Print(strings.Repeat("x", 1<<16))
	case string(config) == "env":
		if os.Getenv("WF_BROWSER_PRIVATE_SECRET") != "" {
			fmt.Print("leaked")
		} else {
			fmt.Print("isolated")
		}
	case string(config) == "fail":
		fmt.Fprintln(os.Stderr, "secret diagnostic must not be returned")
		os.Exit(3)
	case strings.HasPrefix(string(config), "grandchild:"):
		path := strings.TrimPrefix(string(config), "grandchild:")
		child := exec.Command(os.Args[0], "--grandchild", path)
		if err := child.Start(); err != nil {
			os.Exit(4)
		}
		time.Sleep(5 * time.Second)
	default:
		os.Exit(2)
	}
}
