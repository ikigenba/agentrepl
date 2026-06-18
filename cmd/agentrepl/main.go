package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/ikigenba/agentrepl/internal/repl"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, stdoutIsTTY()))
}

func run(args []string, in io.Reader, out, errOut io.Writer, isTTY bool) int {
	opts, err := repl.ParseArgs("agentrepl", args, errOut)
	if err != nil {
		return 1
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(errOut, "startup: home dir: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return repl.Run(ctx, repl.Deps{
		IO: repl.IO{
			In:    in,
			Out:   out,
			Err:   errOut,
			IsTTY: isTTY && os.Getenv("NO_COLOR") == "",
		},
		Getenv: os.Getenv,
		Now:    time.Now,
		LogDir: filepath.Join(home, ".agentkit"),
	}, opts)
}

func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
