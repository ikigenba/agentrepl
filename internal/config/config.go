package config

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ikigenba/agentkit"
	akcatalog "github.com/ikigenba/agentkit/catalog"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

type Target struct {
	Conv         *agentkit.Conversation
	Catalog      []catalog.Provider
	Getenv       func(string) string
	ZaiBaseURL   string
	ReasoningRaw string
	ReasoningKey string
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
			provider, err := p.New(getenv(t), optionsFor(t, p))
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
				models := catalog.Models(p.Name)
				if len(models) > 0 && !hasModel(models, raw) {
					return fmt.Errorf("%w: %q; choose from: %s", catalog.ErrUnknownModel, raw, strings.Join(modelNames(models), ", "))
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
	"temperature": floatField(
		func(c *agentkit.Conversation) **float64 { return &c.Gen.Temperature },
	),
	"top_p": floatField(
		func(c *agentkit.Conversation) **float64 { return &c.Gen.TopP },
	),
	"max_tokens": intField(
		func(c *agentkit.Conversation) *int { return &c.Gen.MaxTokens },
	),
	"effort":          reasoningLevelField("effort"),
	"thinking_budget": reasoningBudgetField("thinking_budget"),
	"thinking_level":  reasoningLevelField("thinking_level"),
	"thinking":        reasoningToggleField("thinking"),
	"max_attempts": intField(
		func(c *agentkit.Conversation) *int { return &c.Retry.MaxAttempts },
	),
	"base_delay": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.BaseDelay },
	),
	"max_delay": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.MaxDelay },
	),
	"max_elapsed": durationField(
		func(c *agentkit.Conversation) *time.Duration { return &c.Retry.MaxElapsed },
	),
	"ignore_retry_after": boolField(
		func(c *agentkit.Conversation) *bool { return &c.Retry.IgnoreRetryAfter },
	),
	"tool_loop_limit": intField(
		func(c *agentkit.Conversation) *int { return &c.MaxToolIterations },
	),
	"base_url": {
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

func reasoningLevelField(key string) field {
	return reasoningField(key, func(display string) (agentkit.ReasoningValue, error) {
		return agentkit.Level(display), nil
	})
}

func reasoningBudgetField(key string) field {
	return reasoningField(key, func(display string) (agentkit.ReasoningValue, error) {
		budget, err := strconv.Atoi(display)
		if err != nil {
			return agentkit.ReasoningValue{}, fmt.Errorf("%w: %s: %v", ErrBadValue, display, err)
		}
		return agentkit.Budget(budget), nil
	})
}

func reasoningToggleField(key string) field {
	return reasoningField(key, func(display string) (agentkit.ReasoningValue, error) {
		switch strings.ToLower(display) {
		case "off":
			return agentkit.DisableReasoning(), nil
		case "on":
			return agentkit.ReasoningValue{}, nil
		default:
			return agentkit.ReasoningValue{}, fmt.Errorf("%w: %s: want on or off", ErrBadValue, display)
		}
	})
}

func reasoningField(key string, parse func(string) (agentkit.ReasoningValue, error)) field {
	return field{
		set: func(t *Target, raw string) error {
			value, display, err := parseReasoningValue(key, raw, parse)
			if err != nil {
				return err
			}
			t.Conv.Gen.Reasoning = value
			t.ReasoningRaw = display
			t.ReasoningKey = key
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.ReasoningRaw == "" || t.ReasoningKey != key {
				return defaultValue
			}
			return t.ReasoningRaw
		},
		reset: resetReasoning,
	}
}

func parseReasoningValue(key, raw string, parse func(string) (agentkit.ReasoningValue, error)) (agentkit.ReasoningValue, string, error) {
	display := strings.TrimSpace(raw)
	if display == "" {
		return agentkit.ReasoningValue{}, "", fmt.Errorf("%w: %s: empty value", ErrBadValue, key)
	}
	value, err := parse(display)
	if err != nil {
		return agentkit.ReasoningValue{}, "", err
	}
	return value, display, nil
}

func resetReasoning(t *Target) error {
	t.Conv.Gen.Reasoning = agentkit.ReasoningValue{}
	t.ReasoningRaw = ""
	t.ReasoningKey = ""
	return nil
}

func getenv(t *Target) func(string) string {
	if t.Getenv != nil {
		return t.Getenv
	}
	return func(string) string { return "" }
}

func optionsFor(t *Target, p catalog.Provider) catalog.Options {
	opts := catalog.Options{Auth: catalog.AuthKey}
	if p.Name == "zai" {
		opts.BaseURL = t.ZaiBaseURL
	}
	return opts
}

func setZaiBaseURL(t *Target, raw string) error {
	if t.Conv.Provider != nil && t.Conv.Provider.Name() == "zai" {
		p, ok := catalog.Lookup(t.Catalog, "zai")
		if !ok {
			return fmt.Errorf("%w: %q", catalog.ErrUnknownProvider, "zai")
		}
		provider, err := p.New(getenv(t), catalog.Options{BaseURL: raw, Auth: catalog.AuthKey})
		if err != nil {
			return fmt.Errorf("provider %q: %w", "zai", err)
		}
		t.Conv.Provider = provider
	}
	t.ZaiBaseURL = raw
	return nil
}

func hasModel(entries []akcatalog.Entry, model string) bool {
	for _, entry := range entries {
		if entry.Model == model {
			return true
		}
	}
	return false
}

func modelNames(entries []akcatalog.Entry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Model
	}
	return names
}
