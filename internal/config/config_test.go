package config

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

func TestSetCoercesEveryKnownKeyToTypedTargetField(t *testing.T) {
	// R-LYK7-Y7ZS
	cases := []struct {
		key    string
		raw    string
		assert func(*testing.T, *Target)
	}{
		{
			key: "provider",
			raw: "test",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Provider == nil || target.Conv.Provider.Name() != "test" {
					t.Fatalf("provider = %v, want named provider", target.Conv.Provider)
				}
			},
		},
		{
			key: "model",
			raw: "loose-model",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Model != "loose-model" {
					t.Fatalf("model = %q, want loose-model", target.Conv.Model)
				}
			},
		},
		{
			key: "system",
			raw: "be concise",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.System != "be concise" {
					t.Fatalf("system = %q, want be concise", target.Conv.System)
				}
			},
		},
		{
			key: "gen.temperature",
			raw: "0.25",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Temperature == nil || *target.Conv.Gen.Temperature != 0.25 {
					t.Fatalf("temperature = %v, want 0.25", target.Conv.Gen.Temperature)
				}
			},
		},
		{
			key: "gen.top_p",
			raw: "0.9",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.TopP == nil || *target.Conv.Gen.TopP != 0.9 {
					t.Fatalf("top_p = %v, want 0.9", target.Conv.Gen.TopP)
				}
			},
		},
		{
			key: "gen.max_tokens",
			raw: "2048",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.MaxTokens != 2048 {
					t.Fatalf("max_tokens = %d, want 2048", target.Conv.Gen.MaxTokens)
				}
			},
		},
		{
			key: "gen.reasoning",
			raw: "xhigh",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Reasoning != agentkit.Level("xhigh") {
					t.Fatalf("reasoning = %v, want xhigh level", target.Conv.Gen.Reasoning)
				}
				if target.ReasoningRaw != "xhigh" {
					t.Fatalf("ReasoningRaw = %q, want xhigh", target.ReasoningRaw)
				}
			},
		},
		{
			key: "retry.max_attempts",
			raw: "5",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxAttempts != 5 {
					t.Fatalf("max_attempts = %d, want 5", target.Conv.Retry.MaxAttempts)
				}
			},
		},
		{
			key: "retry.base_delay",
			raw: "500ms",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.BaseDelay != 500*time.Millisecond {
					t.Fatalf("base_delay = %v, want 500ms", target.Conv.Retry.BaseDelay)
				}
			},
		},
		{
			key: "retry.max_delay",
			raw: "3s",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxDelay != 3*time.Second {
					t.Fatalf("max_delay = %v, want 3s", target.Conv.Retry.MaxDelay)
				}
			},
		},
		{
			key: "retry.max_elapsed",
			raw: "1m30s",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxElapsed != 90*time.Second {
					t.Fatalf("max_elapsed = %v, want 1m30s", target.Conv.Retry.MaxElapsed)
				}
			},
		},
		{
			key: "retry.ignore_retry_after",
			raw: "true",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if !target.Conv.Retry.IgnoreRetryAfter {
					t.Fatal("ignore_retry_after = false, want true")
				}
			},
		},
		{
			key: "tool_loop_limit",
			raw: "12",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.MaxToolIterations != 12 {
					t.Fatalf("tool_loop_limit = %d, want 12", target.Conv.MaxToolIterations)
				}
			},
		},
		{
			key: "zai.base_url",
			raw: "https://api.z.ai/api/coding/paas/v4",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.ZaiBaseURL != "https://api.z.ai/api/coding/paas/v4" {
					t.Fatalf("ZaiBaseURL = %q, want override", target.ZaiBaseURL)
				}
			},
		},
	}

	if len(cases) != len(Keys()) {
		t.Fatalf("coercion cases = %d, want one per key: %v", len(cases), Keys())
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			seen[tc.key] = true
			target := newTarget()
			if err := Set(target, tc.key, tc.raw); err != nil {
				t.Fatalf("Set(%q, %q) returned error: %v", tc.key, tc.raw, err)
			}
			tc.assert(t, target)
		})
	}
	for _, key := range Keys() {
		if !seen[key] {
			t.Fatalf("missing coercion case for %s", key)
		}
	}
}

func TestUnknownKeyWrapsSentinelAndMutatesNothing(t *testing.T) {
	// R-LZS4-BZQH
	target := newTarget()
	target.Conv.System = "original"
	before := *target.Conv

	err := Set(target, "missing.key", "value")
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Set unknown key error = %v, want ErrUnknownKey", err)
	}
	if !strings.Contains(err.Error(), "missing.key") {
		t.Fatalf("unknown key error = %q, want key name", err)
	}
	if !reflect.DeepEqual(before, *target.Conv) {
		t.Fatalf("conversation mutated: before=%#v after=%#v", before, *target.Conv)
	}
}

func TestBadValueWrapsSentinelNamesKeyAndMutatesNothing(t *testing.T) {
	// R-M100-PRH6
	target := newTarget()
	target.Conv.Gen.MaxTokens = 77
	before := *target.Conv

	err := Set(target, "gen.max_tokens", "not-an-int")
	if !errors.Is(err, ErrBadValue) {
		t.Fatalf("Set bad value error = %v, want ErrBadValue", err)
	}
	if !strings.Contains(err.Error(), "gen.max_tokens") || !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("bad value error = %q, want key and parse reason", err)
	}
	if !reflect.DeepEqual(before, *target.Conv) {
		t.Fatalf("conversation mutated: before=%#v after=%#v", before, *target.Conv)
	}
}

func TestDefaultResetsUnsetValuesAndRendersDefault(t *testing.T) {
	// R-M27X-3J7V
	target := newTarget()
	if err := Set(target, "gen.temperature", "0.8"); err != nil {
		t.Fatalf("set temperature: %v", err)
	}
	if err := Set(target, "gen.reasoning", "high"); err != nil {
		t.Fatalf("set reasoning: %v", err)
	}

	if err := Set(target, "gen.temperature", "default"); err != nil {
		t.Fatalf("default temperature: %v", err)
	}
	if err := Set(target, "gen.reasoning", "default"); err != nil {
		t.Fatalf("default reasoning: %v", err)
	}

	if target.Conv.Gen.Temperature != nil {
		t.Fatalf("temperature = %v, want nil", target.Conv.Gen.Temperature)
	}
	if target.Conv.Gen.Reasoning != (agentkit.ReasoningValue{}) {
		t.Fatalf("reasoning = %v, want zero value", target.Conv.Gen.Reasoning)
	}
	if target.ReasoningRaw != "" {
		t.Fatalf("ReasoningRaw = %q, want empty", target.ReasoningRaw)
	}
	for _, key := range []string{"gen.temperature", "gen.reasoning"} {
		got, ok := Get(target, key)
		if !ok || got != "default" {
			t.Fatalf("Get(%q) = %q, %v; want default, true", key, got, ok)
		}
		line := key + "=default"
		if !slices.Contains(Dump(target), line) {
			t.Fatalf("Dump missing %q: %v", line, Dump(target))
		}
	}
}

func TestProviderAndModelCouplingUsesCatalogErrors(t *testing.T) {
	// R-M3FT-HAYK
	t.Run("provider builds through catalog", func(t *testing.T) {
		target := newTarget()
		if err := Set(target, "provider", "test"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if target.Conv.Provider == nil || target.Conv.Provider.Name() != "test" {
			t.Fatalf("provider = %v, want constructed test provider", target.Conv.Provider)
		}
	})

	t.Run("provider unknown", func(t *testing.T) {
		target := newTarget()
		err := Set(target, "provider", "absent")
		if !errors.Is(err, catalog.ErrUnknownProvider) {
			t.Fatalf("Set unknown provider error = %v, want ErrUnknownProvider", err)
		}
		if target.Conv.Provider != nil {
			t.Fatalf("provider = %v, want nil", target.Conv.Provider)
		}
	})

	t.Run("provider missing key", func(t *testing.T) {
		target := newTarget()
		target.Getenv = func(string) string { return "" }
		err := Set(target, "provider", "test")
		if !errors.Is(err, catalog.ErrMissingKey) {
			t.Fatalf("Set provider without key error = %v, want ErrMissingKey", err)
		}
		if !strings.Contains(err.Error(), "TEST_API_KEY") {
			t.Fatalf("missing key error = %q, want env key", err)
		}
		if target.Conv.Provider != nil {
			t.Fatalf("provider = %v, want nil", target.Conv.Provider)
		}
	})

	t.Run("model validates against current provider", func(t *testing.T) {
		target := newTarget()
		if err := Set(target, "provider", "test"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if err := Set(target, "model", "model-a"); err != nil {
			t.Fatalf("Set valid model returned error: %v", err)
		}
		err := Set(target, "model", "model-z")
		if !errors.Is(err, catalog.ErrUnknownModel) {
			t.Fatalf("Set invalid model error = %v, want ErrUnknownModel", err)
		}
		if !strings.Contains(err.Error(), "choose from: model-a, model-b") {
			t.Fatalf("invalid model error = %q, want choices", err)
		}
		if target.Conv.Model != "model-a" {
			t.Fatalf("model = %q, want previous valid model", target.Conv.Model)
		}
	})
}

func TestDumpReturnsAllKeysSortedWithCurrentValues(t *testing.T) {
	// R-M4NP-V2P9
	target := newTarget()
	for _, pair := range []struct {
		key string
		raw string
	}{
		{key: "provider", raw: "test"},
		{key: "model", raw: "model-b"},
		{key: "system", raw: "steady"},
		{key: "gen.temperature", raw: "0.2"},
		{key: "gen.reasoning", raw: "8000"},
		{key: "retry.base_delay", raw: "750ms"},
		{key: "tool_loop_limit", raw: "9"},
		{key: "zai.base_url", raw: "https://api.z.ai/api/coding/paas/v4"},
	} {
		if err := Set(target, pair.key, pair.raw); err != nil {
			t.Fatalf("Set(%q): %v", pair.key, err)
		}
	}

	got := Dump(target)
	want := []string{
		"gen.max_tokens=default",
		"gen.reasoning=8000",
		"gen.temperature=0.2",
		"gen.top_p=default",
		"model=model-b",
		"provider=test",
		"retry.base_delay=750ms",
		"retry.ignore_retry_after=default",
		"retry.max_attempts=default",
		"retry.max_delay=default",
		"retry.max_elapsed=default",
		"system=steady",
		"tool_loop_limit=9",
		"zai.base_url=https://api.z.ai/api/coding/paas/v4",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Dump() = %#v, want %#v", got, want)
	}
	if !slices.IsSorted(got) {
		t.Fatalf("Dump() is not sorted: %v", got)
	}
}

func TestReasoningCoercesNativeShapeModelBlind(t *testing.T) {
	// R-FZCE-VXJL
	cases := []struct {
		name string
		raw  string
		want agentkit.ReasoningValue
	}{
		{name: "off", raw: "off", want: agentkit.DisableReasoning()},
		{name: "disable", raw: "disable", want: agentkit.DisableReasoning()},
		{name: "disabled", raw: "disabled", want: agentkit.DisableReasoning()},
		{name: "negative budget", raw: "-1", want: agentkit.Budget(-1)},
		{name: "zero budget", raw: "0", want: agentkit.Budget(0)},
		{name: "positive budget", raw: "8000", want: agentkit.Budget(8000)},
		{name: "known level", raw: "high", want: agentkit.Level("high")},
		{name: "provider native level", raw: "xhigh", want: agentkit.Level("xhigh")},
		{name: "none stays a level", raw: "none", want: agentkit.Level("none")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := newTarget()
			if err := Set(target, "gen.reasoning", tc.raw); err != nil {
				t.Fatalf("Set gen.reasoning returned error: %v", err)
			}
			if target.Conv.Gen.Reasoning != tc.want {
				t.Fatalf("Reasoning = %v, want %v", target.Conv.Gen.Reasoning, tc.want)
			}
			if target.ReasoningRaw != tc.raw {
				t.Fatalf("ReasoningRaw = %q, want %q", target.ReasoningRaw, tc.raw)
			}
		})
	}
}

func TestReasoningAcceptsNonNativeValuesButRejectsEmpty(t *testing.T) {
	// R-G0KB-9PAA
	target := newTarget()
	if err := Set(target, "gen.reasoning", "made-up-level"); err != nil {
		t.Fatalf("Set non-native gen.reasoning returned error: %v", err)
	}
	if target.Conv.Gen.Reasoning != agentkit.Level("made-up-level") {
		t.Fatalf("Reasoning = %v, want made-up level", target.Conv.Gen.Reasoning)
	}
	if target.ReasoningRaw != "made-up-level" {
		t.Fatalf("ReasoningRaw = %q, want made-up-level", target.ReasoningRaw)
	}

	beforeConv := *target.Conv
	beforeRaw := target.ReasoningRaw
	err := Set(target, "gen.reasoning", "")
	if !errors.Is(err, ErrBadValue) {
		t.Fatalf("Set empty gen.reasoning error = %v, want ErrBadValue", err)
	}
	if !strings.Contains(err.Error(), "gen.reasoning") || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty reasoning error = %q, want key and reason", err)
	}
	if !reflect.DeepEqual(beforeConv, *target.Conv) || target.ReasoningRaw != beforeRaw {
		t.Fatalf("target mutated: before=%#v/%q after=%#v/%q", beforeConv, beforeRaw, *target.Conv, target.ReasoningRaw)
	}
}

func TestReasoningDisplayUsesRawAndDefaultClears(t *testing.T) {
	// R-G304-18RO
	target := newTarget()
	if got, ok := Get(target, "gen.reasoning"); !ok || got != "default" {
		t.Fatalf("unset Get(gen.reasoning) = %q, %v; want default, true", got, ok)
	}
	if err := Set(target, "gen.reasoning", "xhigh"); err != nil {
		t.Fatalf("Set gen.reasoning returned error: %v", err)
	}
	if got, ok := Get(target, "gen.reasoning"); !ok || got != "xhigh" {
		t.Fatalf("Get(gen.reasoning) = %q, %v; want xhigh, true", got, ok)
	}
	if !slices.Contains(Dump(target), "gen.reasoning=xhigh") {
		t.Fatalf("Dump missing gen.reasoning raw value: %v", Dump(target))
	}
	if err := Set(target, "gen.reasoning", "default"); err != nil {
		t.Fatalf("default gen.reasoning returned error: %v", err)
	}
	if target.Conv.Gen.Reasoning != (agentkit.ReasoningValue{}) {
		t.Fatalf("Reasoning = %v, want zero value", target.Conv.Gen.Reasoning)
	}
	if target.ReasoningRaw != "" {
		t.Fatalf("ReasoningRaw = %q, want empty", target.ReasoningRaw)
	}
	if got, ok := Get(target, "gen.reasoning"); !ok || got != "default" {
		t.Fatalf("defaulted Get(gen.reasoning) = %q, %v; want default, true", got, ok)
	}
}

func TestZAIBaseURLStoresAndRebuildsOrderIndependently(t *testing.T) {
	// R-SCS3-DV9R
	override := "https://api.z.ai/api/coding/paas/v4"

	t.Run("base URL before provider", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "zai.base_url", override); err != nil {
			t.Fatalf("Set zai.base_url returned error: %v", err)
		}
		if target.Conv.Provider != nil {
			t.Fatalf("provider = %v, want nil before provider selection", target.Conv.Provider)
		}
		if err := Set(target, "provider", "zai"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != override {
			t.Fatalf("provider baseURL = %q, want %q", got, override)
		}
	})

	t.Run("provider before base URL", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "provider", "zai"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != "" {
			t.Fatalf("initial provider baseURL = %q, want default", got)
		}
		if err := Set(target, "zai.base_url", override); err != nil {
			t.Fatalf("Set zai.base_url returned error: %v", err)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != override {
			t.Fatalf("provider baseURL = %q, want %q", got, override)
		}
	})

	t.Run("default clears and rebuilds active zai", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "zai.base_url", override); err != nil {
			t.Fatalf("Set zai.base_url returned error: %v", err)
		}
		if err := Set(target, "provider", "zai"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if err := Set(target, "zai.base_url", "default"); err != nil {
			t.Fatalf("Set zai.base_url default returned error: %v", err)
		}
		if target.ZaiBaseURL != "" {
			t.Fatalf("ZaiBaseURL = %q, want cleared", target.ZaiBaseURL)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != "" {
			t.Fatalf("provider baseURL = %q, want default", got)
		}
	})

	t.Run("non-zai active provider stores without rebuild", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "provider", "other"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if err := Set(target, "zai.base_url", override); err != nil {
			t.Fatalf("Set zai.base_url returned error: %v", err)
		}
		if target.ZaiBaseURL != override {
			t.Fatalf("ZaiBaseURL = %q, want %q", target.ZaiBaseURL, override)
		}
		if target.Conv.Provider.Name() != "other" {
			t.Fatalf("provider = %q, want other", target.Conv.Provider.Name())
		}
		if err := Set(target, "provider", "zai"); err != nil {
			t.Fatalf("Set provider zai returned error: %v", err)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != override {
			t.Fatalf("zai provider baseURL = %q, want %q", got, override)
		}
	})
}

func TestParsePairAndDirectSetReachIdenticalState(t *testing.T) {
	// R-M5VM-8UFY
	fromFlag := newTarget()
	key, value, err := ParsePair("system=a=b")
	if err != nil {
		t.Fatalf("ParsePair returned error: %v", err)
	}
	if key != "system" || value != "a=b" {
		t.Fatalf("ParsePair = %q, %q; want system, a=b", key, value)
	}
	if err := Set(fromFlag, key, value); err != nil {
		t.Fatalf("Set from ParsePair returned error: %v", err)
	}

	direct := newTarget()
	if err := Set(direct, "system", "a=b"); err != nil {
		t.Fatalf("direct Set returned error: %v", err)
	}

	if !reflect.DeepEqual(Dump(fromFlag), Dump(direct)) {
		t.Fatalf("ParsePair+Set dump = %v, direct Set dump = %v", Dump(fromFlag), Dump(direct))
	}
	if _, _, err := ParsePair("missing-separator"); !errors.Is(err, ErrBadValue) {
		t.Fatalf("ParsePair missing separator error = %v, want ErrBadValue", err)
	}
}

func newTarget() *Target {
	return &Target{
		Conv: &agentkit.Conversation{},
		Catalog: []catalog.Provider{
			{
				Name:   "test",
				EnvKey: "TEST_API_KEY",
				Models: []string{
					"model-a",
					"model-b",
				},
				New: func(_ string, opts catalog.Options) agentkit.Provider {
					return fakeProvider{name: "test", baseURL: opts.BaseURL}
				},
			},
		},
		Getenv: func(key string) string {
			if key == "TEST_API_KEY" {
				return "test-key"
			}
			return ""
		},
	}
}

type fakeProvider struct {
	name    string
	baseURL string
}

func (p fakeProvider) RoundTrip(context.Context, *agentkit.Request) *agentkit.RoundTrip {
	return nil
}

func newZAITarget() *Target {
	return &Target{
		Conv: &agentkit.Conversation{},
		Catalog: []catalog.Provider{
			{
				Name:   "zai",
				EnvKey: "ZAI_API_KEY",
				Models: []string{
					"zai-model",
				},
				New: func(_ string, opts catalog.Options) agentkit.Provider {
					return fakeProvider{name: "zai", baseURL: opts.BaseURL}
				},
			},
			{
				Name:   "other",
				EnvKey: "OTHER_API_KEY",
				Models: []string{
					"other-model",
				},
				New: func(_ string, opts catalog.Options) agentkit.Provider {
					return fakeProvider{name: "other", baseURL: opts.BaseURL}
				},
			},
		},
		Getenv: func(key string) string {
			switch key {
			case "ZAI_API_KEY", "OTHER_API_KEY":
				return "test-key"
			default:
				return ""
			}
		},
	}
}

func providerBaseURL(t *testing.T, provider agentkit.Provider) string {
	t.Helper()
	fake, ok := provider.(fakeProvider)
	if !ok {
		t.Fatalf("provider type = %T, want fakeProvider", provider)
	}
	return fake.baseURL
}

func (p fakeProvider) Name() string {
	return p.name
}

func (p fakeProvider) Pricing(model string) (agentkit.Pricing, bool) {
	switch model {
	case "model-a", "model-b":
		return agentkit.Pricing{}, true
	default:
		return agentkit.Pricing{}, false
	}
}
