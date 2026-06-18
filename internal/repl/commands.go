package repl

import (
	"fmt"
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
	conv   *agentkit.Conversation
	target *config.Target
	cat    []catalog.Provider
	io     IO
	rend   render.Renderer
	color  bool
	getenv func(string) string
	quit   bool
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
				return config.Set(s.target, key, value)
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
					s.rend = render.NewDecorated(s.io.Out, s.color)
				case "raw":
					s.rend = render.NewRaw(s.io.Out)
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
					status := "missing"
					if s.getenv(provider.EnvKey) != "" {
						status = "present"
					}
					s.rend.Notice(fmt.Sprintf("%s %s=%s models=%s", provider.Name, provider.EnvKey, status, strings.Join(provider.Models, ", ")))
				}
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
