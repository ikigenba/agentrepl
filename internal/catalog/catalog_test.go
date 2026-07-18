package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ikigenba/agentkit"
	akcatalog "github.com/ikigenba/agentkit/catalog"
)

func TestDefaultProvidersAndLookup(t *testing.T) {
	// R-4IL7-JPW0
	want := []struct {
		name   string
		envKey string
	}{
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"google", "GEMINI_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
		{"zai", "ZAI_API_KEY"},
	}
	got := Default()
	if len(got) != len(want) {
		t.Fatalf("Default() returned %d providers, want %d", len(got), len(want))
	}
	for i, provider := range got {
		if provider.Name != want[i].name || provider.EnvKey != want[i].envKey {
			t.Errorf("Default()[%d] = %q/%q, want %q/%q", i, provider.Name, provider.EnvKey, want[i].name, want[i].envKey)
		}
	}
	if _, ok := Lookup(got, "unknown"); ok {
		t.Fatal("Lookup(unknown) reported found")
	}
}

func TestDefaultAuthMethods(t *testing.T) {
	// R-4JT3-XHMP
	for _, provider := range Default() {
		want := []AuthMethod{AuthKey}
		if provider.Name == "openai" {
			want = []AuthMethod{AuthSub, AuthKey}
		}
		if !reflect.DeepEqual(provider.Methods, want) {
			t.Errorf("%s methods = %v, want %v", provider.Name, provider.Methods, want)
		}
	}
}

func TestKeyAuthConstructsAndMissingKeyIsClassified(t *testing.T) {
	// R-4L10-B9DE
	for _, provider := range Default() {
		got, err := provider.New(func(string) string { return "secret" }, Options{Auth: AuthKey})
		if err != nil || got == nil {
			t.Errorf("%s key construction = %#v, %v; want non-nil provider", provider.Name, got, err)
		}

		got, err = provider.New(func(string) string { return "" }, Options{Auth: AuthKey})
		if got != nil || !errors.Is(err, ErrMissingKey) || !strings.Contains(err.Error(), provider.EnvKey) {
			t.Errorf("%s missing key = %#v, %v; want nil and ErrMissingKey naming %s", provider.Name, got, err, provider.EnvKey)
		}
	}
}

func TestSubscriptionAuthAndUnsupportedMethods(t *testing.T) {
	// R-4M8W-P143
	provider, _ := Lookup(Default(), "openai")
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"access_token":"header.eyJleHAiOjQxMDI0NDQ4MDAsImh0dHBzOi8vYXBpLm9wZW5haS5jb20vYXV0aCI6eyJjaGF0Z3B0X2FjY291bnRfaWQiOiJhY2N0In19.signature"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := provider.New(func(string) string { return "" }, Options{Auth: AuthSub, AuthFile: path})
	if err != nil || got == nil {
		t.Fatalf("subscription construction = %#v, %v; want non-nil provider", got, err)
	}

	missing := filepath.Join(t.TempDir(), "missing.json")
	got, err = provider.New(func(string) string { return "" }, Options{Auth: AuthSub, AuthFile: missing})
	if got != nil || err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("missing auth file = %#v, %v; want nil error naming path", got, err)
	}
	malformed := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"tokens":`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = provider.New(func(string) string { return "" }, Options{Auth: AuthSub, AuthFile: malformed})
	if got != nil || err == nil || !strings.Contains(err.Error(), malformed) {
		t.Fatalf("malformed auth file = %#v, %v; want nil error naming path", got, err)
	}

	for _, candidate := range Default() {
		got, err := candidate.New(func(string) string { return "secret" }, Options{Auth: "other"})
		if got != nil || !errors.Is(err, ErrAuthUnsupported) || !strings.Contains(err.Error(), candidate.Name) || !strings.Contains(err.Error(), "other") {
			t.Errorf("%s unsupported auth = %#v, %v", candidate.Name, got, err)
		}
	}
}

func TestModelsAreSortedAgentkitChatEntries(t *testing.T) {
	// R-4NGT-2SUS
	for _, provider := range Default() {
		var want []akcatalog.Entry
		for _, entry := range akcatalog.ListByProvider(provider.Name) {
			if entry.Embedding == nil {
				want = append(want, entry)
			}
		}
		if got := Models(provider.Name); !reflect.DeepEqual(got, want) {
			t.Errorf("Models(%q) differs from filtered agentkit catalog", provider.Name)
		}
	}
	foundGLMRoute := false
	for _, entry := range Models("openrouter") {
		if entry.Model == "glm-5.2" {
			foundGLMRoute = true
		}
	}
	if !foundGLMRoute {
		t.Fatal("Models(openrouter) does not include routed glm-5.2")
	}
}

func TestResolveDerivesAndRoutesModels(t *testing.T) {
	// R-4OOP-GKLH
	models := Models("anthropic")
	if len(models) == 0 {
		t.Fatal("anthropic has no chat models")
	}
	provider, wire, entry, ok := Resolve("", models[0].Model)
	if !ok || provider != models[0].Provider || wire != models[0].Model || entry.Model != models[0].Model {
		t.Fatalf("Resolve home = %q/%q/%q/%v", provider, wire, entry.Model, ok)
	}
	provider, wire, entry, ok = Resolve("openrouter", "glm-5.2")
	if !ok || provider != "openrouter" || wire != "z-ai/glm-5.2" || entry.Model != "glm-5.2" {
		t.Fatalf("Resolve GLM route = %q/%q/%q/%v", provider, wire, entry.Model, ok)
	}
	if _, _, _, ok := Resolve("openai", "not-cataloged"); ok {
		t.Fatal("Resolve uncataloged model reported ok")
	}
}

func TestResolveCarriesPricing(t *testing.T) {
	// R-4PWL-UCC6
	var priced akcatalog.Entry
	for _, entry := range Models("openai") {
		if entry.Pricing != nil {
			priced = entry
			break
		}
	}
	if priced.Model == "" {
		t.Fatal("openai catalog contains no priced chat model")
	}
	_, _, got, ok := Resolve("openai", priced.Model)
	if !ok || got.Pricing == nil || !reflect.DeepEqual(got.Pricing, priced.Pricing) {
		t.Fatalf("resolved pricing = %#v/%v, want %#v", got.Pricing, ok, priced.Pricing)
	}
	_, _, got, ok = Resolve("openai", "not-cataloged")
	if ok || !reflect.DeepEqual(got, akcatalog.Entry{}) {
		t.Fatalf("uncataloged entry = %#v/%v, want zero/false", got, ok)
	}
}

func TestBaseURLAppliesToEveryProvider(t *testing.T) {
	// R-4R4I-842V
	transport := &recordingTransport{}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })

	const override = "https://override.example.test/root"
	for _, provider := range Default() {
		models := Models(provider.Name)
		if len(models) == 0 {
			t.Fatalf("%s has no chat models", provider.Name)
		}
		constructed, err := provider.New(func(string) string { return "secret" }, Options{Auth: AuthKey, BaseURL: override})
		if err != nil {
			t.Fatalf("construct %s: %v", provider.Name, err)
		}
		before := len(transport.urls)
		_ = constructed.RoundTrip(context.Background(), &agentkit.Request{Model: models[0].Model})
		if len(transport.urls) != before+1 || !strings.HasPrefix(transport.urls[before], override+"/") {
			t.Errorf("%s request URLs = %v, want override prefix", provider.Name, transport.urls[before:])
		}
	}

	for _, provider := range Default() {
		constructed, err := provider.New(func(string) string { return "secret" }, Options{Auth: AuthKey})
		if err != nil {
			t.Fatalf("construct default %s: %v", provider.Name, err)
		}
		before := len(transport.urls)
		_ = constructed.RoundTrip(context.Background(), &agentkit.Request{Model: Models(provider.Name)[0].Model})
		if len(transport.urls) != before+1 || strings.HasPrefix(transport.urls[before], override) {
			t.Errorf("%s empty BaseURL used override: %v", provider.Name, transport.urls[before:])
		}
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
