package config

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

type Target struct {
	Conv         *agentkit.Conversation
	Catalog      []catalog.Provider
	Getenv       func(string) string
	ZaiBaseURL   string
	ReasoningRaw string
}

var (
	ErrUnknownKey = errors.New("unknown config key")
	ErrBadValue   = errors.New("invalid value for config key")
)

type field struct {
	set   func(*Target, string) error
	get   func(*Target) string
	reset func(*Target) error
}

const defaultValue = "default"

var fields = map[string]field{
	"provider": {
		set: func(t *Target, raw string) error {
			p, ok := catalog.Lookup(t.Catalog, raw)
			if !ok {
				return fmt.Errorf("%w: %q", catalog.ErrUnknownProvider, raw)
			}
			provider, err := p.Build(getenv(t), optionsFor(t, p))
			if err != nil {
				return fmt.Errorf("provider %q: %w", raw, err)
			}
			t.Conv.Provider = provider
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || t.Conv.Provider == nil {
				return defaultValue
			}
			return t.Conv.Provider.Name()
		},
		reset: func(t *Target) error {
			t.Conv.Provider = nil
			return nil
		},
	},
	"model": {
		set: func(t *Target, raw string) error {
			if t.Conv.Provider != nil {
				name := t.Conv.Provider.Name()
				p, ok := catalog.Lookup(t.Catalog, name)
				if !ok {
					return fmt.Errorf("%w: %q", catalog.ErrUnknownProvider, name)
				}
				if !p.HasModel(raw) {
					return fmt.Errorf("%w: %q; choose from: %s", catalog.ErrUnknownModel, raw, strings.Join(p.Models, ", "))
				}
			}
			t.Conv.Model = raw
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || t.Conv.Model == "" {
				return defaultValue
			}
			return t.Conv.Model
		},
		reset: func(t *Target) error {
			t.Conv.Model = ""
			return nil
		},
	},
	"system": {
		set: func(t *Target, raw string) error {
			t.Conv.System = raw
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || t.Conv.System == "" {
				return defaultValue
			}
			return t.Conv.System
		},
		reset: func(t *Target) error {
			t.Conv.System = ""
			return nil
		},
	},
	"gen.temperature": floatField(
		func(c *agentkit.Conversation) **float64 { return &c.Gen.Temperature },
	),
	"gen.top_p": floatField(
		func(c *agentkit.Conversation) **float64 { return &c.Gen.TopP },
	),
	"gen.max_tokens": intField(
		func(c *agentkit.Conversation) *int { return &c.Gen.MaxTokens },
	),
	"gen.reasoning": {
		set: func(t *Target, raw string) error {
			value, display, err := parseReasoning(raw)
			if err != nil {
				return err
			}
			t.Conv.Gen.Reasoning = value
			t.ReasoningRaw = display
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.ReasoningRaw == "" {
				return defaultValue
			}
			return t.ReasoningRaw
		},
		reset: func(t *Target) error {
			t.Conv.Gen.Reasoning = agentkit.ReasoningValue{}
			t.ReasoningRaw = ""
			return nil
		},
	},
	"retry.max_attempts": intField(
		func(c *agentkit.Conversation) *int { return &c.Retry.MaxAttempts },
	),
	"retry.base_delay": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.BaseDelay },
	),
	"retry.max_delay": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.MaxDelay },
	),
	"retry.max_elapsed": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.MaxElapsed },
	),
	"retry.ignore_retry_after": boolField(
		func(c *agentkit.Conversation) *bool { return &c.Retry.IgnoreRetryAfter },
	),
	"tool_loop_limit": intField(
		func(c *agentkit.Conversation) *int { return &c.MaxToolIterations },
	),
	"zai.base_url": {
		set: func(t *Target, raw string) error {
			return setZaiBaseURL(t, raw)
		},
		get: func(t *Target) string {
			if t == nil || t.ZaiBaseURL == "" {
				return defaultValue
			}
			return t.ZaiBaseURL
		},
		reset: func(t *Target) error {
			return setZaiBaseURL(t, "")
		},
	},
}

func Set(t *Target, key, raw string) error {
	f, ok := fields[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
	if t == nil || t.Conv == nil {
		return fmt.Errorf("%w: %s: missing target conversation", ErrBadValue, key)
	}
	if raw == defaultValue {
		return f.reset(t)
	}
	if err := f.set(t, raw); err != nil {
		if errors.Is(err, ErrBadValue) {
			return fmt.Errorf("%s: %w", key, err)
		}
		return err
	}
	return nil
}

func Get(t *Target, key string) (string, bool) {
	f, ok := fields[key]
	if !ok {
		return "", false
	}
	return f.get(t), true
}

func Dump(t *Target) []string {
	keys := Keys()
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		value, _ := Get(t, key)
		out = append(out, key+"="+value)
	}
	return out
}

func Keys() []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func ParsePair(s string) (key, value string, err error) {
	key, value, ok := strings.Cut(s, "=")
	if !ok || key == "" {
		return "", "", fmt.Errorf("%w: expected key=value", ErrBadValue)
	}
	return key, value, nil
}

func floatField(ptr func(*agentkit.Conversation) **float64) field {
	return field{
		set: func(t *Target, raw string) error {
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return fmt.Errorf("%w: %s: %v", ErrBadValue, raw, err)
			}
			*ptr(t.Conv) = &value
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || *ptr(t.Conv) == nil {
				return defaultValue
			}
			return strconv.FormatFloat(**ptr(t.Conv), 'g', -1, 64)
		},
		reset: func(t *Target) error {
			*ptr(t.Conv) = nil
			return nil
		},
	}
}

func intField(ptr func(*agentkit.Conversation) *int) field {
	return field{
		set: func(t *Target, raw string) error {
			value, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("%w: %s: %v", ErrBadValue, raw, err)
			}
			*ptr(t.Conv) = value
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || *ptr(t.Conv) == 0 {
				return defaultValue
			}
			return strconv.Itoa(*ptr(t.Conv))
		},
		reset: func(t *Target) error {
			*ptr(t.Conv) = 0
			return nil
		},
	}
}

func durationField(ptr func(*agentkit.Conversation) *time.Duration) field {
	return field{
		set: func(t *Target, raw string) error {
			value, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("%w: %s: %v", ErrBadValue, raw, err)
			}
			*ptr(t.Conv) = value
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || *ptr(t.Conv) == 0 {
				return defaultValue
			}
			return ptr(t.Conv).String()
		},
		reset: func(t *Target) error {
			*ptr(t.Conv) = 0
			return nil
		},
	}
}

func boolField(ptr func(*agentkit.Conversation) *bool) field {
	return field{
		set: func(t *Target, raw string) error {
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("%w: %s: %v", ErrBadValue, raw, err)
			}
			*ptr(t.Conv) = value
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil || !*ptr(t.Conv) {
				return defaultValue
			}
			return strconv.FormatBool(*ptr(t.Conv))
		},
		reset: func(t *Target) error {
			*ptr(t.Conv) = false
			return nil
		},
	}
}

func parseReasoning(raw string) (agentkit.ReasoningValue, string, error) {
	display := strings.TrimSpace(raw)
	if display == "" {
		return agentkit.ReasoningValue{}, "", fmt.Errorf("%w: gen.reasoning: empty value", ErrBadValue)
	}
	normalized := strings.ToLower(display)
	switch normalized {
	case "off", "disable", "disabled":
		return agentkit.DisableReasoning(), display, nil
	}
	if budget, err := strconv.Atoi(display); err == nil {
		return agentkit.Budget(budget), display, nil
	}
	return agentkit.Level(display), display, nil
}

func getenv(t *Target) func(string) string {
	if t.Getenv != nil {
		return t.Getenv
	}
	return func(string) string { return "" }
}

func optionsFor(t *Target, p catalog.Provider) catalog.Options {
	if p.Name == "zai" {
		return catalog.Options{BaseURL: t.ZaiBaseURL}
	}
	return catalog.Options{}
}

func setZaiBaseURL(t *Target, raw string) error {
	if t.Conv.Provider != nil && t.Conv.Provider.Name() == "zai" {
		p, ok := catalog.Lookup(t.Catalog, "zai")
		if !ok {
			return fmt.Errorf("%w: %q", catalog.ErrUnknownProvider, "zai")
		}
		provider, err := p.Build(getenv(t), catalog.Options{BaseURL: raw})
		if err != nil {
			return fmt.Errorf("provider %q: %w", "zai", err)
		}
		t.Conv.Provider = provider
	}
	t.ZaiBaseURL = raw
	return nil
}
