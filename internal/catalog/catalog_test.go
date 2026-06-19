package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentkit/anthropic"
	"github.com/ikigenba/agentkit/google"
	"github.com/ikigenba/agentkit/openai"
	"github.com/ikigenba/agentkit/zai"
)

type fakeProvider struct {
	name string
}

func (p fakeProvider) RoundTrip(context.Context, *agentkit.Request) *agentkit.RoundTrip {
	return nil
}

func (p fakeProvider) Name() string {
	return p.name
}

func (p fakeProvider) Pricing(string) (agentkit.Pricing, bool) {
	return agentkit.Pricing{}, false
}

func TestDefaultProvidersHaveContractualNamesEnvKeysAndModels(t *testing.T) {
	// R-OVEC-4AWS
	// R-OXU4-VUE6
	got := Default()
	want := []struct {
		name   string
		envKey string
	}{
		{name: "anthropic", envKey: "ANTHROPIC_API_KEY"},
		{name: "google", envKey: "GEMINI_API_KEY"},
		{name: "openai", envKey: "OPENAI_API_KEY"},
		{name: "zai", envKey: "ZAI_API_KEY"},
	}

	if len(got) != len(want) {
		t.Fatalf("Default() returned %d providers, want %d", len(got), len(want))
	}
	for i, provider := range got {
		if provider.Name != want[i].name {
			t.Fatalf("Default()[%d].Name = %q, want %q", i, provider.Name, want[i].name)
		}
		if provider.EnvKey != want[i].envKey {
			t.Fatalf("Default()[%d].EnvKey = %q, want %q", i, provider.EnvKey, want[i].envKey)
		}
		if len(provider.Models) == 0 {
			t.Fatalf("Default()[%d].Models is empty", i)
		}
	}
}

func TestDefaultModelsAreAcceptedByConstructedProviderPricing(t *testing.T) {
	// R-OWM8-I2NH
	for _, provider := range Default() {
		constructed := provider.New("test-key", Options{})
		if constructed == nil {
			t.Fatalf("%s constructor returned nil", provider.Name)
		}
		for _, model := range provider.Models {
			if _, ok := constructed.Pricing(model); !ok {
				t.Fatalf("%s Pricing(%q) ok=false", provider.Name, model)
			}
		}
	}
}

func TestDefaultProvidersSetCredentialBlindReasoningInspectors(t *testing.T) {
	// R-FQT4-7JCQ
	want := map[string]agentkit.ReasoningInspector{
		"anthropic": anthropic.Reasoning,
		"google":    google.Reasoning,
		"openai":    openai.Reasoning,
		"zai":       zai.Reasoning,
	}
	for _, provider := range Default() {
		if provider.Reasoning == nil {
			t.Fatalf("%s Reasoning is nil", provider.Name)
		}
		if provider.Reasoning != want[provider.Name] {
			t.Fatalf("%s Reasoning = %#v, want its package introspector", provider.Name, provider.Reasoning)
		}
		spec, ok := provider.Reasoning.ReasoningSpec(provider.Models[0])
		if !ok {
			t.Fatalf("%s ReasoningSpec(%q) ok=false", provider.Name, provider.Models[0])
		}
		if spec.Term == "" {
			t.Fatalf("%s ReasoningSpec(%q).Term is empty", provider.Name, provider.Models[0])
		}
	}
}

func TestDefaultModelsResolveReasoningSpecs(t *testing.T) {
	// R-FS10-LB3F
	for _, provider := range Default() {
		if provider.Reasoning == nil {
			t.Fatalf("%s Reasoning is nil", provider.Name)
		}
		for _, model := range provider.Models {
			spec, ok := provider.Reasoning.ReasoningSpec(model)
			if !ok {
				t.Fatalf("%s ReasoningSpec(%q) ok=false", provider.Name, model)
			}
			if spec.Term == "" {
				t.Fatalf("%s ReasoningSpec(%q).Term is empty", provider.Name, model)
			}
		}
	}
}

func TestBuildMissingKeyWrapsSentinelNamesEnvAndDoesNotConstruct(t *testing.T) {
	// R-OZ21-9M4V
	calls := 0
	provider := Provider{
		Name:   "test",
		EnvKey: "TEST_API_KEY",
		New: func(string, Options) agentkit.Provider {
			calls++
			return fakeProvider{name: "test"}
		},
	}

	got, err := provider.Build(func(string) string { return "" }, Options{})
	if got != nil {
		t.Fatalf("Build returned provider %#v, want nil", got)
	}
	if !errors.Is(err, ErrMissingKey) {
		t.Fatalf("Build error = %v, want wrapping ErrMissingKey", err)
	}
	if !strings.Contains(err.Error(), provider.EnvKey) {
		t.Fatalf("Build error = %q, want it to name %q", err.Error(), provider.EnvKey)
	}
	if calls != 0 {
		t.Fatalf("constructor called %d times, want 0", calls)
	}
}

func TestBuildConstructsProviderWhenKeyIsPresent(t *testing.T) {
	// R-P09X-NDVK
	var gotKey string
	var gotOptions Options
	wantProvider := fakeProvider{name: "test"}
	provider := Provider{
		Name:   "test",
		EnvKey: "TEST_API_KEY",
		New: func(apiKey string, opts Options) agentkit.Provider {
			gotKey = apiKey
			gotOptions = opts
			return wantProvider
		},
	}

	got, err := provider.Build(func(key string) string {
		if key != provider.EnvKey {
			t.Fatalf("getenv key = %q, want %q", key, provider.EnvKey)
		}
		return "secret"
	}, Options{BaseURL: "https://example.test"})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got == nil {
		t.Fatal("Build returned nil provider")
	}
	if got.Name() != wantProvider.Name() {
		t.Fatalf("Build provider name = %q, want %q", got.Name(), wantProvider.Name())
	}
	if gotKey != "secret" {
		t.Fatalf("constructor apiKey = %q, want %q", gotKey, "secret")
	}
	if gotOptions.BaseURL != "https://example.test" {
		t.Fatalf("constructor options BaseURL = %q, want override", gotOptions.BaseURL)
	}
}

func TestBuildOptionsApplyOnlyToZAIBaseURL(t *testing.T) {
	// R-S94E-8K1O
	transport := &recordingTransport{}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })

	overrideRoot := "https://override.example.test/root"
	cases := []struct {
		name             string
		model            string
		wantOverrideRoot bool
	}{
		{name: "anthropic", model: anthropic.ModelHaiku45},
		{name: "google", model: google.ModelFlash25},
		{name: "openai", model: openai.ModelGPT54Nano},
		{name: "zai", model: zai.ModelGLM46, wantOverrideRoot: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider, ok := Lookup(Default(), tc.name)
			if !ok {
				t.Fatalf("Lookup(%q) ok=false", tc.name)
			}
			constructed, err := provider.Build(func(string) string { return "test-key" }, Options{BaseURL: overrideRoot})
			if err != nil {
				t.Fatalf("Build returned error: %v", err)
			}
			before := len(transport.urls)
			rt := constructed.RoundTrip(context.Background(), &agentkit.Request{Model: tc.model})
			if rt == nil {
				t.Fatal("RoundTrip returned nil")
			}
			if len(transport.urls) != before+1 {
				t.Fatalf("recorded URLs = %v, want one new request", transport.urls)
			}
			got := transport.urls[len(transport.urls)-1]
			gotOverride := strings.HasPrefix(got, overrideRoot+"/")
			if gotOverride != tc.wantOverrideRoot {
				t.Fatalf("%s requested %q, override root used=%v want %v", tc.name, got, gotOverride, tc.wantOverrideRoot)
			}
		})
	}

	provider, ok := Lookup(Default(), "zai")
	if !ok {
		t.Fatal("Lookup(zai) ok=false")
	}
	constructed, err := provider.Build(func(string) string { return "test-key" }, Options{})
	if err != nil {
		t.Fatalf("Build zai default returned error: %v", err)
	}
	before := len(transport.urls)
	_ = constructed.RoundTrip(context.Background(), &agentkit.Request{Model: zai.ModelGLM46})
	if len(transport.urls) != before+1 {
		t.Fatalf("recorded URLs = %v, want one new request", transport.urls)
	}
	if strings.HasPrefix(transport.urls[len(transport.urls)-1], overrideRoot+"/") {
		t.Fatalf("zai default requested override URL %q", transport.urls[len(transport.urls)-1])
	}
}

func TestLookupAndHasModelReportMembership(t *testing.T) {
	// R-P1HU-15M9
	cat := Default()
	provider, ok := Lookup(cat, "anthropic")
	if !ok {
		t.Fatal("Lookup(anthropic) ok=false, want true")
	}
	if _, ok := Lookup(cat, "missing"); ok {
		t.Fatal("Lookup(missing) ok=true, want false")
	}
	if !provider.HasModel(provider.Models[0]) {
		t.Fatalf("HasModel(%q) = false, want true", provider.Models[0])
	}
	if provider.HasModel("not-a-curated-model") {
		t.Fatal("HasModel(not-a-curated-model) = true, want false")
	}
}

type recordingTransport struct {
	urls []string
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.urls = append(t.urls, req.URL.String())
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"stop"}}`)),
		Request:    req,
	}, nil
}
