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
	defer log.Close()

	cat := catalog.Default()
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
			return 1
		}
		if err := config.Set(target, key, value); err != nil {
			fmt.Fprintf(d.IO.Err, "startup: config %q: %v\n", raw, err)
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

	scanner := bufio.NewScanner(d.IO.In)
	for scanner.Scan() {
		line := scanner.Text()
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
			return 0
		}
	}
	if err := scanner.Err(); err != nil {
		state.rend.Error(err)
	}
	return 0
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

func handleTurn(_ context.Context, s *state, _ string) {
	s.rend.Notice("turn execution is not available in this build")
}
