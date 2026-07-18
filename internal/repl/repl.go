package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
	"github.com/ikigenba/agentrepl/internal/config"
	"github.com/ikigenba/agentrepl/internal/render"
	"github.com/ikigenba/agentrepl/internal/session"
	"github.com/ikigenba/agentrepl/internal/tools"
)

// Run opens the session log, builds the conversation, applies startup config,
// and drives the interactive command loop.
func Run(ctx context.Context, d Deps, opts Options) int {
	d = normalizeDeps(d)
	if d.Now == nil {
		fmt.Fprintln(d.IO.Err, "startup: missing clock")
		return 1
	}

	log, _, err := session.Open(d.LogDir, d.Now())
	if err != nil {
		fmt.Fprintf(d.IO.Err, "startup: open session log: %v\n", err)
		return 1
	}

	cat := defaultCatalog()
	color := d.IO.IsTTY && d.Getenv("NO_COLOR") == ""
	rend := newRenderer(d.IO.Out, color, d.IO.IsTTY, opts.Raw)
	conv := &agentkit.Conversation{
		Log:   log,
		Tools: tools.All(),
	}
	target := config.NewTarget(conv, cat, d.Getenv, d.AuthFile)
	for _, raw := range opts.Config {
		key, value, err := config.ParsePair(raw)
		if err != nil {
			fmt.Fprintf(d.IO.Err, "startup: config %q: %v\n", raw, err)
			_ = log.Close()
			return 1
		}
		notice, err := config.Set(target, key, value)
		if err != nil {
			fmt.Fprintf(d.IO.Err, "startup: config %q: %v\n", raw, err)
			_ = log.Close()
			return 1
		}
		if notice != "" {
			rend.Notice(notice)
		}
	}

	state := &state{
		ctx:        ctx,
		conv:       conv,
		target:     target,
		cat:        cat,
		io:         d.IO,
		rend:       rend,
		color:      color,
		getenv:     d.Getenv,
		login:      d.Login,
		liveWaiter: d.Waiter,
		waiter:     activeWaiter(d.Waiter, d.IO.IsTTY, opts.Raw),
	}
	defer log.Close()
	defer conv.Close()
	defer func() {
		state.rend.Summary(conv.TotalUsage(), conv.TotalCost())
	}()

	scanner := bufio.NewScanner(d.IO.In)
	for {
		state.rend.Prompt()
		select {
		case <-ctx.Done():
			state.rend.Notice("interrupted")
			return 130
		case result, ok := <-scanLine(ctx, scanner):
			if !ok {
				return 0
			}
			if result.err != nil {
				state.rend.Error(result.err)
				return 0
			}
			line := result.line
			if strings.TrimSpace(line) == "" {
				continue
			}
			if strings.HasPrefix(line, "/") {
				runCommand(state, line)
			} else {
				if _, err := state.target.Provider(); err != nil {
					state.rend.Error(providerDirective(state.target, err))
					continue
				}
				handleTurn(ctx, state, line)
			}
			if state.quit {
				return state.exitCode
			}
		}
	}
}

func providerDirective(target *config.Target, err error) error {
	provider, ok := catalog.Lookup(target.Cat, target.ProviderName)
	method := catalog.AuthMethod(target.Auth)
	if method == "" && ok && len(provider.Methods) != 0 {
		method = provider.Methods[0]
	}
	switch method {
	case catalog.AuthSub:
		return fmt.Errorf("%w; subscription auth file %q is unavailable: run /login, or set OPENAI_API_KEY then /set auth key, or /set auth_file to an existing Codex login", err, target.AuthFile)
	case catalog.AuthKey:
		if ok && provider.EnvKey != "" {
			return fmt.Errorf("%w; set %s in the environment", err, provider.EnvKey)
		}
	}
	return err
}

type scanResult struct {
	line string
	err  error
}

func scanLine(ctx context.Context, scanner *bufio.Scanner) <-chan scanResult {
	results := make(chan scanResult, 1)
	go func() {
		defer close(results)
		if scanner.Scan() {
			results <- scanResult{line: scanner.Text()}
			return
		}
		if err := scanner.Err(); err != nil {
			select {
			case results <- scanResult{err: err}:
			case <-ctx.Done():
			}
		}
	}()
	return results
}

func normalizeDeps(d Deps) Deps {
	if d.IO.In == nil {
		d.IO.In = strings.NewReader("")
	}
	if d.IO.Out == nil {
		d.IO.Out = io.Discard
	}
	if d.IO.Err == nil {
		d.IO.Err = io.Discard
	}
	if d.Getenv == nil {
		d.Getenv = func(string) string { return "" }
	}
	if d.Waiter == nil {
		d.Waiter = nopWaiter{}
	}
	return d
}

func newRenderer(out io.Writer, color, tty, raw bool) render.Renderer {
	if raw {
		return render.NewRaw(out)
	}
	return render.NewDecorated(out, color, tty)
}

func handleTurn(ctx context.Context, s *state, text string) {
	s.rend.Input(text)
	s.waiter.Start(s.conv.Model)
	defer s.waiter.Stop()
	stream := s.conv.Send(ctx, text)
	stoppedForOutput := false
	stopBeforeOutput := func() {
		if stoppedForOutput {
			return
		}
		s.waiter.Stop()
		stoppedForOutput = true
	}
	for ev := range stream.Events() {
		stopBeforeOutput()
		s.rend.Event(ev)
	}
	if ctx.Err() != nil {
		stopBeforeOutput()
		s.rend.Notice("interrupted")
		s.quit = true
		s.exitCode = 130
		return
	}
	for _, warning := range stream.Warnings() {
		stopBeforeOutput()
		s.rend.Warning(warning)
	}
	if err := stream.Err(); err != nil {
		stopBeforeOutput()
		s.rend.Error(err)
		return
	}
	stopBeforeOutput()
	s.rend.Usage(stream.Usage(), stream.Cost(), s.conv.TotalCost())
}

func activeWaiter(waiter Waiter, tty, raw bool) Waiter {
	if raw || !tty || waiter == nil {
		return nopWaiter{}
	}
	return waiter
}

var defaultCatalog = catalog.Default
