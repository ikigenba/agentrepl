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

const (
	defaultValue    = "default"
	defaultProvider = "openai"
	defaultModel    = "gpt-5.6-sol"
)

type Target struct {
	Conv   *agentkit.Conversation
	Cat    []catalog.Provider
	Getenv func(string) string

	ProviderName     string
	ProviderExplicit bool
	ModelName        string
	Auth             string
	AuthFile         string
	BaseURL          string
	ReasoningRaw     string
	ReasoningKey     string

	authFileDefault string
	built           agentkit.Provider
}

var (
	ErrUnknownKey = errors.New("unknown config key")
	ErrBadValue   = errors.New("invalid value for config key")
)

var keys = []string{
	"auth", "auth_file", "base_delay", "base_url", "effort", "ignore_retry_after",
	"max_attempts", "max_delay", "max_elapsed", "max_tokens", "model", "provider",
	"system", "temperature", "thinking", "thinking_budget", "thinking_level",
	"tool_loop_limit", "top_p",
}

func NewTarget(conv *agentkit.Conversation, cat []catalog.Provider, getenv func(string) string, authFileDefault string) *Target {
	t := &Target{Conv: conv, Cat: cat, Getenv: getenv, ProviderName: defaultProvider, AuthFile: authFileDefault, authFileDefault: authFileDefault}
	_, wire, entry, ok := catalog.Resolve("", defaultModel)
	if ok {
		t.ModelName = defaultModel
		t.ProviderName = entry.Provider
		conv.Model = wire
		conv.Pricing = entry.Pricing
	}
	return t
}

func (t *Target) Provider() (agentkit.Provider, error) {
	if t == nil || t.Conv == nil {
		return nil, fmt.Errorf("%w: provider: missing target conversation", ErrBadValue)
	}
	if t.built != nil {
		return t.built, nil
	}
	p, ok := catalog.Lookup(t.Cat, t.ProviderName)
	if !ok {
		return nil, unknownProvider(t, t.ProviderName)
	}
	built, err := p.New(t.getenv(), catalog.Options{BaseURL: t.BaseURL, Auth: catalog.AuthMethod(t.Auth), AuthFile: t.AuthFile})
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", t.ProviderName, err)
	}
	t.built = built
	t.Conv.Provider = built
	return built, nil
}

func Set(t *Target, key, raw string) (notice string, err error) {
	if !slices.Contains(keys, key) {
		return "", fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
	if t == nil || t.Conv == nil {
		return "", fmt.Errorf("%w: %s: missing target conversation", ErrBadValue, key)
	}
	if raw == "" {
		return "", badValue(key, "empty value")
	}
	if raw == defaultValue {
		return "", reset(t, key)
	}

	switch key {
	case "provider":
		if _, ok := catalog.Lookup(t.Cat, raw); !ok {
			return "", unknownProvider(t, raw)
		}
		t.ProviderName, t.ProviderExplicit = raw, true
		t.invalidate()
	case "model":
		return setModel(t, raw)
	case "auth":
		method := catalog.AuthMethod(raw)
		if method != catalog.AuthKey && method != catalog.AuthSub {
			return "", badValue(key, "want key or sub")
		}
		p, ok := catalog.Lookup(t.Cat, t.ProviderName)
		if !ok {
			return "", unknownProvider(t, t.ProviderName)
		}
		if !slices.Contains(p.Methods, method) {
			return "", badValue(key, fmt.Sprintf("provider %s does not support method %s", p.Name, method))
		}
		t.Auth = raw
		t.invalidate()
	case "auth_file":
		t.AuthFile = raw
		t.invalidate()
	case "base_url":
		t.BaseURL = raw
		t.invalidate()
	case "system":
		t.Conv.System = raw
	case "temperature":
		v, e := strconv.ParseFloat(raw, 64)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Gen.Temperature = &v
	case "top_p":
		v, e := strconv.ParseFloat(raw, 64)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Gen.TopP = &v
	case "max_tokens":
		v, e := strconv.Atoi(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Gen.MaxTokens = v
	case "max_attempts":
		v, e := strconv.Atoi(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Retry.MaxAttempts = v
	case "base_delay":
		v, e := time.ParseDuration(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Retry.BaseDelay = v
	case "max_delay":
		v, e := time.ParseDuration(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Retry.MaxDelay = v
	case "max_elapsed":
		v, e := time.ParseDuration(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Retry.MaxElapsed = v
	case "ignore_retry_after":
		v, e := strconv.ParseBool(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.Retry.IgnoreRetryAfter = v
	case "tool_loop_limit":
		v, e := strconv.Atoi(raw)
		if e != nil {
			return "", badValue(key, e.Error())
		}
		t.Conv.MaxToolIterations = v
	case "effort", "thinking_level", "thinking_budget", "thinking":
		return "", setReasoning(t, key, raw)
	}
	return "", nil
}

func setModel(t *Target, raw string) (string, error) {
	provider := ""
	if t.ProviderExplicit {
		provider = t.ProviderName
	}
	route, wire, entry, ok := catalog.Resolve(provider, raw)
	if !ok && !t.ProviderExplicit {
		return "", fmt.Errorf("%w: %q not in the agentkit catalog; set provider explicitly to send it anyway", catalog.ErrUnknownModel, raw)
	}
	if ok {
		t.ModelName, t.Conv.Model, t.Conv.Pricing = raw, wire, entry.Pricing
		if !t.ProviderExplicit {
			t.ProviderName = route
			t.invalidate()
		}
		return "", nil
	}
	t.ModelName, t.Conv.Model, t.Conv.Pricing = raw, raw, nil
	return "model not in catalog: no pricing (cost reports 0), reasoning unchecked", nil
}

func setReasoning(t *Target, key, raw string) error {
	value := agentkit.ReasoningValue{}
	var err error
	switch key {
	case "effort", "thinking_level":
		value = agentkit.Level(raw)
	case "thinking_budget":
		var n int
		n, err = strconv.Atoi(raw)
		if err == nil {
			value = agentkit.Budget(n)
		}
	case "thinking":
		switch raw {
		case "on":
		case "off":
			value = agentkit.DisableReasoning()
		default:
			err = errors.New("want on or off")
		}
	}
	if err != nil {
		return badValue(key, err.Error())
	}
	if entry, ok := akcatalog.Lookup(t.ModelName); ok && entry.Reasoning != nil {
		accepted, spec, _ := akcatalog.Check(t.ModelName, value)
		if !accepted {
			return badValue(key, fmt.Sprintf("not accepted for %s; accepted values: %s", t.ModelName, acceptedValues(spec)))
		}
	}
	t.Conv.Gen.Reasoning, t.ReasoningRaw, t.ReasoningKey = value, raw, key
	return nil
}

func acceptedValues(spec *akcatalog.ReasoningSpec) string {
	if spec == nil {
		return "none"
	}
	parts := append([]string(nil), spec.Levels...)
	if spec.Kind == akcatalog.ReasoningRange {
		parts = append(parts, fmt.Sprintf("%d..%d", spec.Min, spec.Max))
	}
	for _, sentinel := range spec.Sentinels {
		parts = append(parts, strconv.Itoa(sentinel.Value))
	}
	if spec.CanDisable {
		parts = append(parts, "off")
	}
	return strings.Join(parts, ", ")
}

func reset(t *Target, key string) error {
	switch key {
	case "provider":
		t.ProviderName, t.ProviderExplicit = defaultProvider, false
		t.invalidate()
	case "model":
		_, err := setModel(t, defaultModel)
		return err
	case "auth":
		t.Auth = ""
		t.invalidate()
	case "auth_file":
		t.AuthFile = t.authFileDefault
		t.invalidate()
	case "base_url":
		t.BaseURL = ""
		t.invalidate()
	case "system":
		t.Conv.System = ""
	case "temperature":
		t.Conv.Gen.Temperature = nil
	case "top_p":
		t.Conv.Gen.TopP = nil
	case "max_tokens":
		t.Conv.Gen.MaxTokens = 0
	case "max_attempts":
		t.Conv.Retry.MaxAttempts = 0
	case "base_delay":
		t.Conv.Retry.BaseDelay = 0
	case "max_delay":
		t.Conv.Retry.MaxDelay = 0
	case "max_elapsed":
		t.Conv.Retry.MaxElapsed = 0
	case "ignore_retry_after":
		t.Conv.Retry.IgnoreRetryAfter = false
	case "tool_loop_limit":
		t.Conv.MaxToolIterations = 0
	case "effort", "thinking_level", "thinking_budget", "thinking":
		t.Conv.Gen.Reasoning = agentkit.ReasoningValue{}
		t.ReasoningRaw, t.ReasoningKey = "", ""
	}
	return nil
}

func Get(t *Target, key string) (string, bool) {
	if !slices.Contains(keys, key) {
		return "", false
	}
	if t == nil || t.Conv == nil {
		return defaultValue, true
	}
	switch key {
	case "provider":
		return t.ProviderName, true
	case "model":
		return t.ModelName, true
	case "auth":
		if t.Auth != "" {
			return t.Auth, true
		}
	case "auth_file":
		if t.AuthFile != "" {
			return t.AuthFile, true
		}
	case "base_url":
		if t.BaseURL != "" {
			return t.BaseURL, true
		}
	case "system":
		if t.Conv.System != "" {
			return t.Conv.System, true
		}
	case "temperature":
		if t.Conv.Gen.Temperature != nil {
			return strconv.FormatFloat(*t.Conv.Gen.Temperature, 'g', -1, 64), true
		}
	case "top_p":
		if t.Conv.Gen.TopP != nil {
			return strconv.FormatFloat(*t.Conv.Gen.TopP, 'g', -1, 64), true
		}
	case "max_tokens":
		if t.Conv.Gen.MaxTokens != 0 {
			return strconv.Itoa(t.Conv.Gen.MaxTokens), true
		}
	case "max_attempts":
		if t.Conv.Retry.MaxAttempts != 0 {
			return strconv.Itoa(t.Conv.Retry.MaxAttempts), true
		}
	case "base_delay":
		if t.Conv.Retry.BaseDelay != 0 {
			return t.Conv.Retry.BaseDelay.String(), true
		}
	case "max_delay":
		if t.Conv.Retry.MaxDelay != 0 {
			return t.Conv.Retry.MaxDelay.String(), true
		}
	case "max_elapsed":
		if t.Conv.Retry.MaxElapsed != 0 {
			return t.Conv.Retry.MaxElapsed.String(), true
		}
	case "ignore_retry_after":
		if t.Conv.Retry.IgnoreRetryAfter {
			return "true", true
		}
	case "tool_loop_limit":
		if t.Conv.MaxToolIterations != 0 {
			return strconv.Itoa(t.Conv.MaxToolIterations), true
		}
	case "effort", "thinking_level", "thinking_budget", "thinking":
		if t.ReasoningKey == key && t.ReasoningRaw != "" {
			return t.ReasoningRaw, true
		}
	}
	return defaultValue, true
}

func Dump(t *Target) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		v, _ := Get(t, key)
		out = append(out, key+"="+v)
	}
	return out
}
func Keys() []string { return append([]string(nil), keys...) }

func ParsePair(s string) (key, value string, err error) {
	key, value, ok := strings.Cut(s, "=")
	if !ok || key == "" || value == "" {
		return "", "", badValue("config", "expected key=value")
	}
	return key, value, nil
}

func (t *Target) invalidate() { t.built = nil; t.Conv.Provider = nil }
func (t *Target) getenv() func(string) string {
	if t.Getenv != nil {
		return t.Getenv
	}
	return func(string) string { return "" }
}
func badValue(key, reason string) error { return fmt.Errorf("%w: %s: %s", ErrBadValue, key, reason) }
func unknownProvider(t *Target, name string) error {
	names := make([]string, len(t.Cat))
	for i, p := range t.Cat {
		names[i] = p.Name
	}
	slices.Sort(names)
	return fmt.Errorf("%w: %q; choose from: %s", catalog.ErrUnknownProvider, name, strings.Join(names, ", "))
}
