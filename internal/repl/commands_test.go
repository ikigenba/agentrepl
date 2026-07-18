package repl

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentrepl/internal/catalog"
)

type fakeLoginFlow struct {
	url      string
	complete func(context.Context, string, string) error
}

func (f *fakeLoginFlow) AuthorizeURL() string { return f.url }

func (f *fakeLoginFlow) Complete(ctx context.Context, path, redirectURL string) error {
	if f.complete == nil {
		return nil
	}
	return f.complete(ctx, path, redirectURL)
}

func TestLoginCompletesFromREPLScannerAndInvalidatesProvider(t *testing.T) {
	// R-LRTQ-4X56
	provider := newScriptedProvider(successRound("first", usageOne()), successRound("second", usageOne()))
	var builds int
	originalCatalog := defaultCatalog
	defaultCatalog = func() []catalog.Provider {
		return []catalog.Provider{{
			Name: "openai", Methods: []catalog.AuthMethod{catalog.AuthSub},
			New: func(func(string) string, catalog.Options) (agentkit.Provider, error) {
				builds++
				return provider, nil
			},
		}}
	}
	t.Cleanup(func() { defaultCatalog = originalCatalog })

	authFile := filepath.Join(t.TempDir(), "current.json")
	defaultAuthFile := filepath.Join(t.TempDir(), "default.json")
	beginCalls := 0
	completeCalls := 0
	var out, errOut bytes.Buffer
	code := Run(context.Background(), Deps{
		IO:       IO{In: strings.NewReader("first\n/login\n  http://localhost:1455/callback?code=ok  \nsecond\n/exit\n"), Out: &out, Err: &errOut},
		Getenv:   func(string) string { return "" },
		Now:      func() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC) },
		LogDir:   t.TempDir(),
		AuthFile: defaultAuthFile,
		BeginLogin: func() (LoginFlow, error) {
			beginCalls++
			return &fakeLoginFlow{
				url: "https://auth.example/authorize?challenge=abc",
				complete: func(_ context.Context, path, redirectURL string) error {
					completeCalls++
					if path != authFile || redirectURL != "http://localhost:1455/callback?code=ok" {
						t.Fatalf("Complete(path, URL) = (%q, %q), want current auth file and trimmed paste", path, redirectURL)
					}
					return nil
				},
			}, nil
		},
	}, Options{Config: []string{"auth_file=" + authFile}})
	if code != 0 || beginCalls != 1 || completeCalls != 1 {
		t.Fatalf("Run code/begin/complete = %d/%d/%d, stderr %q", code, beginCalls, completeCalls, errOut.String())
	}
	if builds != 2 || len(provider.requests) != 2 {
		t.Fatalf("provider builds/requests = %d/%d, want invalidated build and next turn", builds, len(provider.requests))
	}
	for _, want := range []string{
		"https://auth.example/authorize?challenge=abc",
		"dead http://localhost:1455/ page",
		"Copy the full URL from the address bar",
		"paste it back here",
		"Paste the full redirect URL",
		"subscription login saved to " + authFile,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}
}

func TestLoginEmptyOrEOFCancelsWithoutCompleting(t *testing.T) {
	// R-LT1M-IOVV
	tests := []struct {
		name       string
		script     string
		wantFollow string
	}{
		{name: "empty line continues", script: "/login\n  \n/help\n/exit\n", wantFollow: "config keys:"},
		{name: "EOF exits normally", script: "/login\n", wantFollow: "subscription login cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completeCalls := 0
			var out, errOut bytes.Buffer
			code := Run(context.Background(), Deps{
				IO:       IO{In: strings.NewReader(tt.script), Out: &out, Err: &errOut},
				Getenv:   func(string) string { return "" },
				Now:      func() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC) },
				LogDir:   t.TempDir(),
				AuthFile: filepath.Join(t.TempDir(), "auth.json"),
				BeginLogin: func() (LoginFlow, error) {
					return &fakeLoginFlow{url: "https://auth.example", complete: func(context.Context, string, string) error {
						completeCalls++
						return nil
					}}, nil
				},
			}, Options{})
			if code != 0 || completeCalls != 0 {
				t.Fatalf("Run code/complete = %d/%d, stderr %q", code, completeCalls, errOut.String())
			}
			if !strings.Contains(out.String(), "subscription login cancelled") || !strings.Contains(out.String(), tt.wantFollow) {
				t.Fatalf("stdout = %q, want cancellation and %q", out.String(), tt.wantFollow)
			}
		})
	}
}

func TestLoginErrorsAreNonFatalAndKeepCachedProvider(t *testing.T) {
	// R-LU9I-WGMK
	tests := []struct {
		name      string
		script    string
		begin     func() (LoginFlow, error)
		wantError string
	}{
		{
			name:   "begin error",
			script: "first\n/login\nsecond\n/help\n/exit\n",
			begin: func() (LoginFlow, error) {
				return nil, errors.New("browser launch setup failed")
			},
			wantError: "browser launch setup failed",
		},
		{
			name:   "complete error",
			script: "first\n/login\nhttp://localhost:1455/callback\nsecond\n/help\n/exit\n",
			begin: func() (LoginFlow, error) {
				return &fakeLoginFlow{url: "https://auth.example", complete: func(context.Context, string, string) error {
					return errors.New("expected the full redirect URL")
				}}, nil
			},
			wantError: "expected the full redirect URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newScriptedProvider(successRound("first", usageOne()), successRound("second", usageOne()))
			builds := 0
			originalCatalog := defaultCatalog
			defaultCatalog = func() []catalog.Provider {
				return []catalog.Provider{{
					Name: "openai", Methods: []catalog.AuthMethod{catalog.AuthSub},
					New: func(func(string) string, catalog.Options) (agentkit.Provider, error) {
						builds++
						return provider, nil
					},
				}}
			}
			t.Cleanup(func() { defaultCatalog = originalCatalog })

			var out, errOut bytes.Buffer
			code := Run(context.Background(), Deps{
				IO:         IO{In: strings.NewReader(tt.script), Out: &out, Err: &errOut},
				Getenv:     func(string) string { return "" },
				Now:        func() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC) },
				LogDir:     t.TempDir(),
				AuthFile:   filepath.Join(t.TempDir(), "auth.json"),
				BeginLogin: tt.begin,
			}, Options{})
			if code != 0 || builds != 1 || len(provider.requests) != 2 {
				t.Fatalf("Run code/builds/requests = %d/%d/%d, stderr %q", code, builds, len(provider.requests), errOut.String())
			}
			if !strings.Contains(out.String(), tt.wantError) || !strings.Contains(out.String(), "config keys:") {
				t.Fatalf("stdout = %q, want underlying error %q and continued help", out.String(), tt.wantError)
			}
		})
	}
}
