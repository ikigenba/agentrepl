package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
	akcatalog "github.com/ikigenba/agentkit/catalog"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

func TestNewTargetSeedsResolvedDefaultsWithoutConstructing(t *testing.T) {
	// R-4X80-4YSC
	builds := 0
	conv := &agentkit.Conversation{}
	target := NewTarget(conv, testCatalog(&builds, nil), func(string) string { return "" }, "/auth/default.json")
	entry, _ := akcatalog.Lookup(defaultModel)
	if target.ProviderName != "openai" || target.ProviderExplicit || target.ModelName != defaultModel || target.Auth != "" || target.AuthFile != "/auth/default.json" {
		t.Fatalf("seeded target = %#v", target)
	}
	if conv.Model != defaultModel || !reflect.DeepEqual(conv.Pricing, entry.Pricing) || conv.Provider != nil || builds != 0 {
		t.Fatalf("conversation seed model/pricing/provider/builds = %q/%v/%v/%d", conv.Model, conv.Pricing, conv.Provider, builds)
	}
	want := []string{"auth=default", "auth_file=/auth/default.json", "base_delay=default", "base_url=default", "effort=default", "ignore_retry_after=default", "max_attempts=default", "max_delay=default", "max_elapsed=default", "max_tokens=default", "model=gpt-5.6-sol", "provider=openai", "system=default", "temperature=default", "thinking=default", "thinking_budget=default", "thinking_level=default", "tool_loop_limit=default", "top_p=default"}
	if !reflect.DeepEqual(Dump(target), want) {
		t.Fatalf("Dump = %v, want %v", Dump(target), want)
	}
}

func TestDefaultRestoresSeededAndZeroState(t *testing.T) {
	// R-4YFW-IQJ1
	target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/default-auth")
	for _, pair := range [][2]string{{"provider", "zai"}, {"model", "free"}, {"auth", "key"}, {"auth_file", "/other"}, {"base_url", "https://example.test"}, {"system", "x"}, {"temperature", "1"}, {"top_p", ".5"}, {"max_tokens", "2"}, {"max_attempts", "3"}, {"base_delay", "1s"}, {"max_delay", "2s"}, {"max_elapsed", "3s"}, {"ignore_retry_after", "true"}, {"tool_loop_limit", "4"}, {"effort", "wild"}} {
		if _, err := Set(target, pair[0], pair[1]); err != nil {
			t.Fatalf("Set %v: %v", pair, err)
		}
	}
	for _, key := range Keys() {
		if _, err := Set(target, key, "default"); err != nil {
			t.Fatalf("reset %s: %v", key, err)
		}
	}
	if target.ProviderName != defaultProvider || target.ProviderExplicit || target.ModelName != defaultModel || target.Auth != "" || target.AuthFile != "/default-auth" {
		t.Fatalf("seeded fields not restored: %#v", target)
	}
	for _, line := range Dump(target) {
		if strings.HasPrefix(line, "provider=") || strings.HasPrefix(line, "model=") || strings.HasPrefix(line, "auth_file=") {
			continue
		}
		if !strings.HasSuffix(line, "=default") {
			t.Fatalf("non-seed value did not reset: %s", line)
		}
	}
}

func TestBadValuesNameKeyWrapSentinelAndMutateNothing(t *testing.T) {
	// R-4ZNS-WI9Q
	for _, key := range Keys() {
		t.Run(key, func(t *testing.T) {
			target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/auth")
			before := Dump(target)
			_, err := Set(target, key, "")
			if !errors.Is(err, ErrBadValue) || !strings.Contains(err.Error(), key) || !reflect.DeepEqual(Dump(target), before) {
				t.Fatalf("Set empty error/state = %v/%v, want ErrBadValue and unchanged", err, Dump(target))
			}
		})
	}
	for _, pair := range [][2]string{{"thinking_budget", "many"}, {"thinking", "maybe"}} {
		target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/auth")
		if _, err := Set(target, pair[0], pair[1]); !errors.Is(err, ErrBadValue) {
			t.Fatalf("Set %v = %v, want ErrBadValue", pair, err)
		}
	}
}

func TestImplicitProviderModelResolutionAndUnknownRejection(t *testing.T) {
	// R-50VP-AA0F
	target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/auth")
	_, wire, entry, _ := catalog.Resolve("", "glm-5.2")
	if _, err := Set(target, "model", "glm-5.2"); err != nil {
		t.Fatal(err)
	}
	if target.ProviderName != entry.Provider || target.ProviderExplicit || target.Conv.Model != wire || !reflect.DeepEqual(target.Conv.Pricing, entry.Pricing) {
		t.Fatalf("resolved target = %#v", target)
	}
	before := Dump(target)
	if _, err := Set(target, "model", "uncataloged"); !errors.Is(err, catalog.ErrUnknownModel) || !reflect.DeepEqual(Dump(target), before) {
		t.Fatalf("unknown model error/state = %v/%v", err, Dump(target))
	}
}

func TestExplicitProviderCatalogRoutingAndFreeFlow(t *testing.T) {
	// R-523L-O1R4
	target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/auth")
	if _, err := Set(target, "provider", "openrouter"); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(target, "model", "glm-5.2"); err != nil {
		t.Fatal(err)
	}
	if target.Conv.Model != "z-ai/glm-5.2" || target.Conv.Pricing == nil {
		t.Fatalf("routed model/pricing = %q/%v", target.Conv.Model, target.Conv.Pricing)
	}
	if got, _ := Get(target, "model"); got != "glm-5.2" {
		t.Fatalf("display model = %q", got)
	}
	notice, err := Set(target, "model", "vendor-preview")
	if err != nil || notice == "" || target.Conv.Model != "vendor-preview" || target.Conv.Pricing != nil {
		t.Fatalf("free flow = %q/%v/%q/%v", notice, err, target.Conv.Model, target.Conv.Pricing)
	}
	if !strings.Contains(strings.Join(Dump(target), "\n"), "model=vendor-preview") {
		t.Fatalf("Dump = %v", Dump(target))
	}
}

func TestAuthValidationAndProviderDefaults(t *testing.T) {
	// R-4TKA-ZNK9
	var effective []catalog.AuthMethod
	cat := testCatalog(new(int), func(p catalog.Provider, opts catalog.Options) (agentkit.Provider, error) {
		method := opts.Auth
		if method == "" {
			method = p.Methods[0]
		}
		effective = append(effective, method)
		return fakeProvider(p.Name), nil
	})
	target := NewTarget(&agentkit.Conversation{}, cat, nil, "/auth")
	if _, err := Set(target, "auth", "token"); !errors.Is(err, ErrBadValue) {
		t.Fatalf("invalid auth = %v", err)
	}
	if _, err := Set(target, "provider", "zai"); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(target, "auth", "sub"); !errors.Is(err, ErrBadValue) || !strings.Contains(err.Error(), "zai") || !strings.Contains(err.Error(), "sub") || target.Auth != "" {
		t.Fatalf("unsupported auth = %v state %q", err, target.Auth)
	}
	if _, err := target.Provider(); err != nil {
		t.Fatal(err)
	}
	target2 := NewTarget(&agentkit.Conversation{}, cat, nil, "/auth")
	if _, err := target2.Provider(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(effective, []catalog.AuthMethod{catalog.AuthKey, catalog.AuthSub}) {
		t.Fatalf("effective methods = %v", effective)
	}
}

func TestLazyProviderUsesOptionsCachesAndInvalidates(t *testing.T) {
	// R-4W03-R71N
	// R-54JE-FL8I
	builds := 0
	var got catalog.Options
	cat := testCatalog(&builds, func(p catalog.Provider, opts catalog.Options) (agentkit.Provider, error) {
		got = opts
		if opts.AuthFile == "bad" {
			return nil, errors.New("bad auth file")
		}
		return &countedProvider{name: p.Name, n: builds}, nil
	})
	target := NewTarget(&agentkit.Conversation{}, cat, nil, "/auth.json")
	if _, err := Set(target, "base_url", "https://example.test"); err != nil {
		t.Fatal(err)
	}
	first, err := target.Provider()
	if err != nil {
		t.Fatal(err)
	}
	second, err := target.Provider()
	if err != nil || first != second || builds != 1 {
		t.Fatalf("cache = %v/%v builds=%d", first, second, builds)
	}
	if got.BaseURL != "https://example.test" || got.Auth != "" || got.AuthFile != "/auth.json" || target.Conv.Provider != first {
		t.Fatalf("options/assignment = %#v/%v", got, target.Conv.Provider)
	}
	for _, pair := range [][2]string{{"provider", "openai"}, {"auth", "key"}, {"auth_file", "/next"}, {"base_url", "https://next.test"}} {
		if _, err := Set(target, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
		if _, err := target.Provider(); err != nil {
			t.Fatal(err)
		}
	}
	if builds != 5 {
		t.Fatalf("builds after four invalidations = %d", builds)
	}
	if _, err := Set(target, "auth_file", "bad"); err != nil {
		t.Fatal(err)
	}
	if _, err := target.Provider(); err == nil || !strings.Contains(err.Error(), "bad auth file") {
		t.Fatalf("construction error = %v", err)
	}
}

func TestReasoningCatalogGateAndFreeFlowPassThrough(t *testing.T) {
	// R-53BI-1THT
	target := NewTarget(&agentkit.Conversation{}, testCatalog(new(int), nil), nil, "/auth")
	entry, _ := akcatalog.Lookup(defaultModel)
	if entry.Reasoning == nil || len(entry.Reasoning.Levels) == 0 {
		t.Fatal("default model lacks enum reasoning spec")
	}
	accepted := entry.Reasoning.Levels[0]
	if _, err := Set(target, "effort", accepted); err != nil {
		t.Fatal(err)
	}
	before := target.Conv.Gen.Reasoning
	if _, err := Set(target, "effort", "not-native"); !errors.Is(err, ErrBadValue) || !strings.Contains(err.Error(), "accepted values") || target.Conv.Gen.Reasoning != before {
		t.Fatalf("catalog rejection = %v reasoning=%v", err, target.Conv.Gen.Reasoning)
	}
	if _, err := Set(target, "provider", "openrouter"); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(target, "model", "free-preview"); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(target, "effort", "not-native"); err != nil || target.Conv.Gen.Reasoning != agentkit.Level("not-native") {
		t.Fatalf("free-flow reasoning = %v/%v", err, target.Conv.Gen.Reasoning)
	}
}

func TestParsePairAndKeys(t *testing.T) {
	key, value, err := ParsePair("system=a=b")
	if err != nil || key != "system" || value != "a=b" {
		t.Fatalf("ParsePair = %q/%q/%v", key, value, err)
	}
	if _, _, err := ParsePair("missing"); !errors.Is(err, ErrBadValue) {
		t.Fatalf("bad pair = %v", err)
	}
	if !reflect.DeepEqual(Keys(), keys) {
		t.Fatalf("Keys = %v", Keys())
	}
}

func testCatalog(builds *int, hook func(catalog.Provider, catalog.Options) (agentkit.Provider, error)) []catalog.Provider {
	methods := map[string][]catalog.AuthMethod{"openai": {catalog.AuthSub, catalog.AuthKey}, "anthropic": {catalog.AuthKey}, "google": {catalog.AuthKey}, "openrouter": {catalog.AuthKey}, "zai": {catalog.AuthKey}}
	out := make([]catalog.Provider, 0, len(methods))
	for _, name := range []string{"anthropic", "google", "openai", "openrouter", "zai"} {
		p := catalog.Provider{Name: name, EnvKey: strings.ToUpper(name) + "_API_KEY", Methods: methods[name]}
		p.New = func(_ func(string) string, opts catalog.Options) (agentkit.Provider, error) {
			*builds++
			if hook != nil {
				return hook(p, opts)
			}
			return fakeProvider(p.Name), nil
		}
		out = append(out, p)
	}
	return out
}

type fakeProvider string

func (p fakeProvider) Name() string                                                     { return string(p) }
func (p fakeProvider) RoundTrip(context.Context, *agentkit.Request) *agentkit.RoundTrip { return nil }
func (p fakeProvider) Pricing(string) (agentkit.Pricing, bool)                          { return agentkit.Pricing{}, false }

type countedProvider struct {
	name string
	n    int
}

func (p *countedProvider) Name() string { return fmt.Sprintf("%s-%d", p.name, p.n) }
func (p *countedProvider) RoundTrip(context.Context, *agentkit.Request) *agentkit.RoundTrip {
	return nil
}
func (p *countedProvider) Pricing(string) (agentkit.Pricing, bool) { return agentkit.Pricing{}, false }
