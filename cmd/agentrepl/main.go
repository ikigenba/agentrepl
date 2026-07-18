package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/ikigenba/agentrepl/internal/render"
	"github.com/ikigenba/agentrepl/internal/repl"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, stdoutIsTTY()))
}

func run(args []string, in io.Reader, out, errOut io.Writer, isTTY bool) int {
	parseOut := errOut
	if hasHelpFlag(args) {
		parseOut = out
	}
	opts, err := repl.ParseArgs("agentrepl", args, parseOut)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(errOut, "startup: home dir: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var waiter repl.Waiter
	if isTTY {
		waiter = render.NewLiveWaiter(out, os.Getenv("NO_COLOR") == "")
	}

	return repl.Run(ctx, repl.Deps{
		IO: repl.IO{
			In:    in,
			Out:   out,
			Err:   errOut,
			IsTTY: isTTY,
		},
		Getenv:   os.Getenv,
		Now:      time.Now,
		Waiter:   waiter,
		LogDir:   filepath.Join(home, ".agentkit"),
		AuthFile: filepath.Join(home, ".agentrepl", "auth.json"),
	}, opts)
}

func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "-help" || arg == "--help" {
			return true
		}
	}
	return false
}
