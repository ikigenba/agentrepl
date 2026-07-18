package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
	"github.com/ikigenba/agentrepl/internal/config"
	"github.com/ikigenba/agentrepl/internal/render"
)

type command struct {
	summary string
	usage   string
	run     func(s *state, args string) error
}

type state struct {
	ctx        context.Context
	conv       *agentkit.Conversation
	target     *config.Target
	cat        []catalog.Provider
	io         IO
	rend       render.Renderer
	color      bool
	getenv     func(string) string
	beginLogin func() (LoginFlow, error)
	scanner    *bufio.Scanner
	liveWaiter Waiter
	waiter     Waiter
	quit       bool
	exitCode   int
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"set": {
			summary: "set config",
			usage:   "/set <key> <value>",
			run: func(s *state, args string) error {
				key, value, ok := strings.Cut(strings.TrimSpace(args), " ")
				if !ok || key == "" {
					return fmt.Errorf("usage: /set <key> <value>")
				}
				notice, err := config.Set(s.target, key, value)
				if notice != "" {
					s.rend.Notice(notice)
				}
				return err
			},
		},
		"get": {
			summary: "show one config value",
			usage:   "/get <key>",
			run: func(s *state, args string) error {
				key := strings.TrimSpace(args)
				value, ok := config.Get(s.target, key)
				if !ok {
					return fmt.Errorf("%w: %s", config.ErrUnknownKey, key)
				}
				s.rend.Notice(key + "=" + value)
				return nil
			},
		},
		"dump": {
			summary: "show all config values",
			usage:   "/dump",
			run: func(s *state, _ string) error {
				for _, line := range config.Dump(s.target) {
					s.rend.Notice(line)
				}
				return nil
			},
		},
		"usage": {
			summary: "show cumulative usage",
			usage:   "/usage",
			run: func(s *state, _ string) error {
				s.rend.Summary(s.conv.TotalUsage(), s.conv.TotalCost())
				return nil
			},
		},
		"clear": {
			summary: "clear conversation history",
			usage:   "/clear",
			run: func(s *state, _ string) error {
				s.conv.History = nil
				s.rend.Notice("conversation history cleared")
				return nil
			},
		},
		"render": {
			summary: "switch renderer",
			usage:   "/render <decorated|raw>",
			run: func(s *state, args string) error {
				switch strings.TrimSpace(args) {
				case "decorated":
					s.rend = render.NewDecorated(s.io.Out, s.color, s.io.IsTTY)
					s.waiter = activeWaiter(s.liveWaiter, s.io.IsTTY, false)
				case "raw":
					s.rend = render.NewRaw(s.io.Out)
					s.waiter = activeWaiter(s.liveWaiter, s.io.IsTTY, true)
				default:
					return fmt.Errorf("usage: /render <decorated|raw>")
				}
				s.rend.Notice("render mode changed")
				return nil
			},
		},
		"providers": {
			summary: "list providers",
			usage:   "/providers",
			run: func(s *state, _ string) error {
				for _, provider := range s.cat {
					for _, method := range provider.Methods {
						switch method {
						case catalog.AuthKey:
							status := "missing"
							if s.getenv(provider.EnvKey) != "" {
								status = "present"
							}
							s.rend.Notice(fmt.Sprintf("%s key %s=%s", provider.Name, provider.EnvKey, status))
						case catalog.AuthSub:
							status := "missing"
							if info, err := os.Stat(s.target.AuthFile); err == nil && !info.IsDir() {
								status = "present"
							}
							s.rend.Notice(fmt.Sprintf("%s sub %s=%s", provider.Name, s.target.AuthFile, status))
						}
					}
				}
				return nil
			},
		},
		"login": {
			summary: "log in with an OpenAI subscription",
			usage:   "/login",
			run: func(s *state, _ string) error {
				if s.beginLogin == nil {
					return fmt.Errorf("subscription login is unavailable")
				}
				flow, err := s.beginLogin()
				if err != nil {
					return fmt.Errorf("begin subscription login: %w", err)
				}
				s.rend.Notice("Open this URL in any browser and authorize: " + flow.AuthorizeURL())
				s.rend.Notice("The browser will land on a dead http://localhost:1455/ page; this is expected.")
				s.rend.Notice("Copy the full URL from the address bar and paste it back here.")
				s.rend.Notice("Paste the full redirect URL (empty line cancels):")
				if s.scanner == nil {
					return fmt.Errorf("subscription login input is unavailable")
				}
				result, ok := <-scanLine(s.ctx, s.scanner)
				if !ok || (result.err == nil && strings.TrimSpace(result.line) == "") {
					s.rend.Notice("subscription login cancelled")
					return nil
				}
				if result.err != nil {
					return fmt.Errorf("read subscription login redirect: %w", result.err)
				}
				redirectURL := strings.TrimSpace(result.line)
				if err := flow.Complete(s.ctx, s.target.AuthFile, redirectURL); err != nil {
					return fmt.Errorf("complete subscription login: %w", err)
				}
				if _, err := config.Set(s.target, "auth_file", s.target.AuthFile); err != nil {
					return fmt.Errorf("invalidate provider after login: %w", err)
				}
				s.rend.Notice("subscription login saved to " + s.target.AuthFile)
				return nil
			},
		},
		"help": {
			summary: "show help",
			usage:   "/help",
			run: func(s *state, _ string) error {
				for _, name := range commandNames() {
					cmd := commands[name]
					s.rend.Notice(fmt.Sprintf("/%s - %s (%s)", name, cmd.summary, cmd.usage))
				}
				s.rend.Notice("config keys: " + strings.Join(config.Keys(), ", "))
				return nil
			},
		},
		"exit": {
			summary: "exit",
			usage:   "/exit",
			run: func(s *state, _ string) error {
				s.quit = true
				return nil
			},
		},
		"quit": {
			summary: "exit",
			usage:   "/quit",
			run: func(s *state, _ string) error {
				s.quit = true
				return nil
			},
		},
	}
}

func runCommand(s *state, line string) {
	name, args, _ := strings.Cut(strings.TrimPrefix(line, "/"), " ")
	cmd, ok := commands[name]
	if !ok {
		s.rend.Error(fmt.Errorf("unknown command: /%s", name))
		return
	}
	if err := cmd.run(s, strings.TrimSpace(args)); err != nil {
		s.rend.Error(err)
	}
}

func commandNames() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
