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
	conv := &agentkit.Conversation{
		Log:   log,
		Tools: tools.All(),
	}
	target := &config.Target{
		Conv:    conv,
		Catalog: cat,
		Getenv:  d.Getenv,
	}
	for _, raw := range opts.Config {
		key, value, err := config.ParsePair(raw)
		if err != nil {
			fmt.Fprintf(d.IO.Err, "startup: config %q: %v\n", raw, err)
			_ = log.Close()
			return 1
		}
		if err := config.Set(target, key, value); err != nil {
			fmt.Fprintf(d.IO.Err, "startup: config %q: %v\n", raw, err)
			_ = log.Close()
			return 1
		}
	}

	color := d.IO.IsTTY && d.Getenv("NO_COLOR") == ""
	state := &state{
		conv:   conv,
		target: target,
		cat:    cat,
		io:     d.IO,
		rend:   newRenderer(d.IO.Out, color, opts.Raw),
		color:  color,
		getenv: d.Getenv,
	}
	defer log.Close()
	defer conv.Close()
	defer func() {
		state.rend.Summary(conv.TotalUsage(), conv.TotalCost())
	}()

	lines := scanLines(ctx, d.IO.In)
	for {
		select {
		case <-ctx.Done():
			state.rend.Notice("interrupted")
			return 130
		case result, ok := <-lines:
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
			} else if state.conv.Provider == nil || state.conv.Model == "" {
				state.rend.Notice("set a provider and model first - e.g. `/set provider anthropic` then `/set model ...`")
			} else {
				handleTurn(ctx, state, line)
			}
			if state.quit {
				return state.exitCode
			}
		}
	}
}

type scanResult struct {
	line string
	err  error
}

func scanLines(ctx context.Context, in io.Reader) <-chan scanResult {
	results := make(chan scanResult, 1)
	go func() {
		defer close(results)
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			select {
			case results <- scanResult{line: scanner.Text()}:
			case <-ctx.Done():
				return
			}
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
	return d
}

func newRenderer(out io.Writer, color bool, raw bool) render.Renderer {
	if raw {
		return render.NewRaw(out)
	}
	return render.NewDecorated(out, color)
}

func handleTurn(ctx context.Context, s *state, text string) {
	s.rend.Prompt(text)
	stream := s.conv.Send(ctx, text)
	for ev := range stream.Events() {
		s.rend.Event(ev)
	}
	if ctx.Err() != nil {
		s.rend.Notice("interrupted")
		s.quit = true
		s.exitCode = 130
		return
	}
	for _, warning := range stream.Warnings() {
		s.rend.Warning(warning)
	}
	if err := stream.Err(); err != nil {
		s.rend.Error(err)
		return
	}
	s.rend.Usage(stream.Usage(), stream.Cost(), s.conv.TotalCost())
}

var defaultCatalog = catalog.Default
