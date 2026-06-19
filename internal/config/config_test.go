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
			key: "temperature",
			raw: "0.25",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Temperature == nil || *target.Conv.Gen.Temperature != 0.25 {
					t.Fatalf("temperature = %v, want 0.25", target.Conv.Gen.Temperature)
				}
			},
		},
		{
			key: "top_p",
			raw: "0.9",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.TopP == nil || *target.Conv.Gen.TopP != 0.9 {
					t.Fatalf("top_p = %v, want 0.9", target.Conv.Gen.TopP)
				}
			},
		},
		{
			key: "max_tokens",
			raw: "2048",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.MaxTokens != 2048 {
					t.Fatalf("max_tokens = %d, want 2048", target.Conv.Gen.MaxTokens)
				}
			},
		},
		{
			key: "effort",
			raw: "xhigh",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Reasoning != agentkit.Level("xhigh") {
					t.Fatalf("reasoning = %v, want xhigh level", target.Conv.Gen.Reasoning)
				}
				if target.ReasoningRaw != "xhigh" || target.ReasoningKey != "effort" {
					t.Fatalf("reasoning display = %q/%q, want xhigh/effort", target.ReasoningRaw, target.ReasoningKey)
				}
			},
		},
		{
			key: "thinking_budget",
			raw: "8000",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Reasoning != agentkit.Budget(8000) {
					t.Fatalf("reasoning = %v, want budget 8000", target.Conv.Gen.Reasoning)
				}
				if target.ReasoningRaw != "8000" || target.ReasoningKey != "thinking_budget" {
					t.Fatalf("reasoning display = %q/%q, want 8000/thinking_budget", target.ReasoningRaw, target.ReasoningKey)
				}
			},
		},
		{
			key: "thinking_level",
			raw: "high",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Reasoning != agentkit.Level("high") {
					t.Fatalf("reasoning = %v, want high level", target.Conv.Gen.Reasoning)
				}
				if target.ReasoningRaw != "high" || target.ReasoningKey != "thinking_level" {
					t.Fatalf("reasoning display = %q/%q, want high/thinking_level", target.ReasoningRaw, target.ReasoningKey)
				}
			},
		},
		{
			key: "thinking",
			raw: "off",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Gen.Reasoning != agentkit.DisableReasoning() {
					t.Fatalf("reasoning = %v, want disable", target.Conv.Gen.Reasoning)
				}
				if target.ReasoningRaw != "off" || target.ReasoningKey != "thinking" {
					t.Fatalf("reasoning display = %q/%q, want off/thinking", target.ReasoningRaw, target.ReasoningKey)
				}
			},
		},
		{
			key: "max_attempts",
			raw: "5",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxAttempts != 5 {
					t.Fatalf("max_attempts = %d, want 5", target.Conv.Retry.MaxAttempts)
				}
			},
		},
		{
			key: "base_delay",
			raw: "500ms",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.BaseDelay != 500*time.Millisecond {
					t.Fatalf("base_delay = %v, want 500ms", target.Conv.Retry.BaseDelay)
				}
			},
		},
		{
			key: "max_delay",
			raw: "3s",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxDelay != 3*time.Second {
					t.Fatalf("max_delay = %v, want 3s", target.Conv.Retry.MaxDelay)
				}
			},
		},
		{
			key: "max_elapsed",
			raw: "1m30s",
			assert: func(t *testing.T, target *Target) {
				t.Helper()
				if target.Conv.Retry.MaxElapsed != 90*time.Second {
					t.Fatalf("max_elapsed = %v, want 1m30s", target.Conv.Retry.MaxElapsed)
				}
			},
		},
		{
			key: "ignore_retry_after",
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
			key: "base_url",
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
	for _, key := range []string{"missing.key", "gen.temperature", "gen.reasoning", "retry.max_attempts", "zai.base_url"} {
		t.Run(key, func(t *testing.T) {
			target := newTarget()
			target.Conv.System = "original"
			before := *target.Conv

			err := Set(target, key, "value")
			if !errors.Is(err, ErrUnknownKey) {
				t.Fatalf("Set unknown key error = %v, want ErrUnknownKey", err)
			}
			if !strings.Contains(err.Error(), key) {
				t.Fatalf("unknown key error = %q, want key name", err)
			}
			if !reflect.DeepEqual(before, *target.Conv) {
				t.Fatalf("conversation mutated: before=%#v after=%#v", before, *target.Conv)
			}
		})
	}
}

func TestBadValueWrapsSentinelNamesKeyAndMutatesNothing(t *testing.T) {
	// R-M100-PRH6
	for _, tc := range []struct {
		key    string
		raw    string
		reason string
	}{
		{key: "temperature", raw: "not-a-float", reason: "invalid syntax"},
		{key: "max_tokens", raw: "not-an-int", reason: "invalid syntax"},
		{key: "base_delay", raw: "soon", reason: "invalid duration"},
		{key: "ignore_retry_after", raw: "maybe", reason: "invalid syntax"},
		{key: "tool_loop_limit", raw: "none", reason: "invalid syntax"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			target := newTarget()
			target.Conv.System = "unchanged"
			before := *target.Conv

			err := Set(target, tc.key, tc.raw)
			if !errors.Is(err, ErrBadValue) {
				t.Fatalf("Set bad value error = %v, want ErrBadValue", err)
			}
			if !strings.Contains(err.Error(), tc.key) || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("bad value error = %q, want key and parse reason", err)
			}
			if !reflect.DeepEqual(before, *target.Conv) {
				t.Fatalf("conversation mutated: before=%#v after=%#v", before, *target.Conv)
			}
		})
	}
}

func TestDefaultResetsUnsetValuesAndRendersDefault(t *testing.T) {
	// R-M27X-3J7V
	target := newTarget()
	if err := Set(target, "temperature", "0.8"); err != nil {
		t.Fatalf("set temperature: %v", err)
	}
	if err := Set(target, "effort", "high"); err != nil {
		t.Fatalf("set reasoning: %v", err)
	}

	if err := Set(target, "temperature", "default"); err != nil {
		t.Fatalf("default temperature: %v", err)
	}
	if err := Set(target, "thinking_budget", "default"); err != nil {
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
	if target.ReasoningKey != "" {
		t.Fatalf("ReasoningKey = %q, want empty", target.ReasoningKey)
	}
	for _, key := range []string{"temperature", "effort", "thinking_budget"} {
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
		{key: "temperature", raw: "0.2"},
		{key: "thinking_budget", raw: "8000"},
		{key: "base_delay", raw: "750ms"},
		{key: "tool_loop_limit", raw: "9"},
		{key: "base_url", raw: "https://api.z.ai/api/coding/paas/v4"},
	} {
		if err := Set(target, pair.key, pair.raw); err != nil {
			t.Fatalf("Set(%q): %v", pair.key, err)
		}
	}

	got := Dump(target)
	want := []string{
		"base_delay=750ms",
		"base_url=https://api.z.ai/api/coding/paas/v4",
		"effort=default",
		"ignore_retry_after=default",
		"max_attempts=default",
		"max_delay=default",
		"max_elapsed=default",
		"max_tokens=default",
		"model=model-b",
		"provider=test",
		"system=steady",
		"temperature=0.2",
		"thinking=default",
		"thinking_budget=8000",
		"thinking_level=default",
		"tool_loop_limit=9",
		"top_p=default",
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
		key  string
		raw  string
		want agentkit.ReasoningValue
	}{
		{name: "effort known level", key: "effort", raw: "high", want: agentkit.Level("high")},
		{name: "effort provider native level", key: "effort", raw: "xhigh", want: agentkit.Level("xhigh")},
		{name: "effort none stays a level", key: "effort", raw: "none", want: agentkit.Level("none")},
		{name: "thinking level", key: "thinking_level", raw: "high", want: agentkit.Level("high")},
		{name: "thinking level xhigh", key: "thinking_level", raw: "xhigh", want: agentkit.Level("xhigh")},
		{name: "thinking level none", key: "thinking_level", raw: "none", want: agentkit.Level("none")},
		{name: "negative budget", key: "thinking_budget", raw: "-1", want: agentkit.Budget(-1)},
		{name: "zero budget", key: "thinking_budget", raw: "0", want: agentkit.Budget(0)},
		{name: "positive budget", key: "thinking_budget", raw: "8000", want: agentkit.Budget(8000)},
		{name: "thinking off", key: "thinking", raw: "off", want: agentkit.DisableReasoning()},
		{name: "thinking on", key: "thinking", raw: "on", want: agentkit.ReasoningValue{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := newTarget()
			if err := Set(target, tc.key, tc.raw); err != nil {
				t.Fatalf("Set(%q, %q) returned error: %v", tc.key, tc.raw, err)
			}
			if target.Conv.Gen.Reasoning != tc.want {
				t.Fatalf("Reasoning = %v, want %v", target.Conv.Gen.Reasoning, tc.want)
			}
			if target.ReasoningRaw != tc.raw || target.ReasoningKey != tc.key {
				t.Fatalf("reasoning display = %q/%q, want %q/%q", target.ReasoningRaw, target.ReasoningKey, tc.raw, tc.key)
			}
		})
	}
}

func TestReasoningAcceptsNonNativeValuesButRejectsStructurallyUnusableInput(t *testing.T) {
	// R-G0KB-9PAA
	target := newTarget()
	if err := Set(target, "effort", "made-up-level"); err != nil {
		t.Fatalf("Set non-native effort returned error: %v", err)
	}
	if target.Conv.Gen.Reasoning != agentkit.Level("made-up-level") {
		t.Fatalf("Reasoning = %v, want made-up level", target.Conv.Gen.Reasoning)
	}
	if target.ReasoningRaw != "made-up-level" || target.ReasoningKey != "effort" {
		t.Fatalf("reasoning display = %q/%q, want made-up-level/effort", target.ReasoningRaw, target.ReasoningKey)
	}

	beforeConv := *target.Conv
	beforeRaw := target.ReasoningRaw
	beforeKey := target.ReasoningKey
	for _, tc := range []struct {
		key    string
		raw    string
		reason string
	}{
		{key: "effort", raw: "", reason: "empty"},
		{key: "thinking_level", raw: "", reason: "empty"},
		{key: "thinking_budget", raw: "", reason: "empty"},
		{key: "thinking_budget", raw: "high", reason: "invalid syntax"},
		{key: "thinking", raw: "", reason: "empty"},
		{key: "thinking", raw: "maybe", reason: "want on or off"},
	} {
		t.Run(tc.key+"="+tc.raw, func(t *testing.T) {
			err := Set(target, tc.key, tc.raw)
			if !errors.Is(err, ErrBadValue) {
				t.Fatalf("Set(%q, %q) error = %v, want ErrBadValue", tc.key, tc.raw, err)
			}
			if !strings.Contains(err.Error(), tc.key) || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("reasoning error = %q, want key and reason", err)
			}
			if !reflect.DeepEqual(beforeConv, *target.Conv) || target.ReasoningRaw != beforeRaw || target.ReasoningKey != beforeKey {
				t.Fatalf("target mutated: before=%#v/%q/%q after=%#v/%q/%q", beforeConv, beforeRaw, beforeKey, *target.Conv, target.ReasoningRaw, target.ReasoningKey)
			}
		})
	}
}

func TestReasoningDisplayUsesRawAndDefaultClears(t *testing.T) {
	// R-G304-18RO
	target := newTarget()
	for _, key := range []string{"effort", "thinking_budget", "thinking_level", "thinking"} {
		if got, ok := Get(target, key); !ok || got != "default" {
			t.Fatalf("unset Get(%q) = %q, %v; want default, true", key, got, ok)
		}
	}
	if err := Set(target, "effort", "xhigh"); err != nil {
		t.Fatalf("Set effort returned error: %v", err)
	}
	if got, ok := Get(target, "effort"); !ok || got != "xhigh" {
		t.Fatalf("Get(effort) = %q, %v; want xhigh, true", got, ok)
	}
	for _, key := range []string{"thinking_budget", "thinking_level", "thinking"} {
		if got, ok := Get(target, key); !ok || got != "default" {
			t.Fatalf("Get(%q) = %q, %v; want default, true", key, got, ok)
		}
	}
	if !slices.Contains(Dump(target), "effort=xhigh") || slices.Contains(Dump(target), "thinking_budget=xhigh") {
		t.Fatalf("Dump did not render reasoning only under effort: %v", Dump(target))
	}

	if err := Set(target, "thinking_budget", "8000"); err != nil {
		t.Fatalf("Set thinking_budget returned error: %v", err)
	}
	if got, ok := Get(target, "thinking_budget"); !ok || got != "8000" {
		t.Fatalf("Get(thinking_budget) = %q, %v; want 8000, true", got, ok)
	}
	if got, ok := Get(target, "effort"); !ok || got != "default" {
		t.Fatalf("Get(effort) = %q, %v; want default after overwrite, true", got, ok)
	}

	if err := Set(target, "thinking", "default"); err != nil {
		t.Fatalf("default thinking returned error: %v", err)
	}
	if target.Conv.Gen.Reasoning != (agentkit.ReasoningValue{}) {
		t.Fatalf("Reasoning = %v, want zero value", target.Conv.Gen.Reasoning)
	}
	if target.ReasoningRaw != "" {
		t.Fatalf("ReasoningRaw = %q, want empty", target.ReasoningRaw)
	}
	if target.ReasoningKey != "" {
		t.Fatalf("ReasoningKey = %q, want empty", target.ReasoningKey)
	}
	for _, key := range []string{"effort", "thinking_budget", "thinking_level", "thinking"} {
		if got, ok := Get(target, key); !ok || got != "default" {
			t.Fatalf("defaulted Get(%q) = %q, %v; want default, true", key, got, ok)
		}
	}
}

func TestZAIBaseURLStoresAndRebuildsOrderIndependently(t *testing.T) {
	// R-SCS3-DV9R
	override := "https://api.z.ai/api/coding/paas/v4"

	t.Run("base URL before provider", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "base_url", override); err != nil {
			t.Fatalf("Set base_url returned error: %v", err)
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
		if err := Set(target, "base_url", override); err != nil {
			t.Fatalf("Set base_url returned error: %v", err)
		}
		if got := providerBaseURL(t, target.Conv.Provider); got != override {
			t.Fatalf("provider baseURL = %q, want %q", got, override)
		}
	})

	t.Run("default clears and rebuilds active zai", func(t *testing.T) {
		target := newZAITarget()
		if err := Set(target, "base_url", override); err != nil {
			t.Fatalf("Set base_url returned error: %v", err)
		}
		if err := Set(target, "provider", "zai"); err != nil {
			t.Fatalf("Set provider returned error: %v", err)
		}
		if err := Set(target, "base_url", "default"); err != nil {
			t.Fatalf("Set base_url default returned error: %v", err)
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
		if err := Set(target, "base_url", override); err != nil {
			t.Fatalf("Set base_url returned error: %v", err)
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
