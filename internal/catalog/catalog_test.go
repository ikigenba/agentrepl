package catalog

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
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
		constructed := provider.New("test-key")
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

func TestBuildMissingKeyWrapsSentinelNamesEnvAndDoesNotConstruct(t *testing.T) {
	// R-OZ21-9M4V
	calls := 0
	provider := Provider{
		Name:   "test",
		EnvKey: "TEST_API_KEY",
		New: func(string) agentkit.Provider {
			calls++
			return fakeProvider{name: "test"}
		},
	}

	got, err := provider.Build(func(string) string { return "" })
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
	wantProvider := fakeProvider{name: "test"}
	provider := Provider{
		Name:   "test",
		EnvKey: "TEST_API_KEY",
		New: func(apiKey string) agentkit.Provider {
			gotKey = apiKey
			return wantProvider
		},
	}

	got, err := provider.Build(func(key string) string {
		if key != provider.EnvKey {
			t.Fatalf("getenv key = %q, want %q", key, provider.EnvKey)
		}
		return "secret"
	})
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
