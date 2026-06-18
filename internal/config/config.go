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
	Conv    *agentkit.Conversation
	Catalog []catalog.Provider
	Getenv  func(string) string
}

var (
	ErrUnknownKey = errors.New("unknown config key")
	ErrBadValue   = errors.New("invalid value for config key")
)

type field struct {
	set   func(*Target, string) error
	get   func(*Target) string
	reset func(*Target)
}

const defaultValue = "default"

var fields = map[string]field{
	"provider": {
		set: func(t *Target, raw string) error {
			p, ok := catalog.Lookup(t.Catalog, raw)
			if !ok {
				return fmt.Errorf("%w: %q", catalog.ErrUnknownProvider, raw)
			}
			provider, err := p.Build(getenv(t))
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
		reset: func(t *Target) { t.Conv.Provider = nil },
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
		reset: func(t *Target) { t.Conv.Model = "" },
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
		reset: func(t *Target) { t.Conv.System = "" },
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
			effort, ok := parseReasoning(raw)
			if !ok {
				return fmt.Errorf("%w: gen.reasoning: expected off, minimal, low, medium, high, max, or default", ErrBadValue)
			}
			t.Conv.Gen.Reasoning = effort
			return nil
		},
		get: func(t *Target) string {
			if t == nil || t.Conv == nil {
				return defaultValue
			}
			return formatReasoning(t.Conv.Gen.Reasoning)
		},
		reset: func(t *Target) { t.Conv.Gen.Reasoning = agentkit.EffortDefault },
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
		f.reset(t)
		return nil
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
		reset: func(t *Target) { *ptr(t.Conv) = nil },
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
		reset: func(t *Target) { *ptr(t.Conv) = 0 },
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
		reset: func(t *Target) { *ptr(t.Conv) = 0 },
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
		reset: func(t *Target) { *ptr(t.Conv) = false },
	}
}

func parseReasoning(raw string) (agentkit.ReasoningEffort, bool) {
	switch raw {
	case "default":
		return agentkit.EffortDefault, true
	case "off":
		return agentkit.EffortOff, true
	case "minimal":
		return agentkit.EffortMinimal, true
	case "low":
		return agentkit.EffortLow, true
	case "medium":
		return agentkit.EffortMedium, true
	case "high":
		return agentkit.EffortHigh, true
	case "max":
		return agentkit.EffortMax, true
	default:
		return agentkit.EffortDefault, false
	}
}

func formatReasoning(effort agentkit.ReasoningEffort) string {
	switch effort {
	case agentkit.EffortOff:
		return "off"
	case agentkit.EffortMinimal:
		return "minimal"
	case agentkit.EffortLow:
		return "low"
	case agentkit.EffortMedium:
		return "medium"
	case agentkit.EffortHigh:
		return "high"
	case agentkit.EffortMax:
		return "max"
	default:
		return defaultValue
	}
}

func getenv(t *Target) func(string) string {
	if t.Getenv != nil {
		return t.Getenv
	}
	return func(string) string { return "" }
}
