package catalog

import (
	"errors"
	"fmt"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentkit/anthropic"
	akcatalog "github.com/ikigenba/agentkit/catalog"
	"github.com/ikigenba/agentkit/google"
	"github.com/ikigenba/agentkit/openai"
	"github.com/ikigenba/agentkit/openai/subscription"
	"github.com/ikigenba/agentkit/openrouter"
	"github.com/ikigenba/agentkit/zai"
)

type AuthMethod string

const (
	AuthKey AuthMethod = "key"
	AuthSub AuthMethod = "sub"
)

type Options struct {
	BaseURL  string
	Auth     AuthMethod
	AuthFile string
}

type Provider struct {
	Name    string
	EnvKey  string
	Methods []AuthMethod
	New     func(getenv func(string) string, opts Options) (agentkit.Provider, error)
}

var (
	ErrUnknownProvider = errors.New("unknown provider")
	ErrUnknownModel    = errors.New("unknown model")
	ErrMissingKey      = errors.New("missing API key")
	ErrAuthUnsupported = errors.New("auth method not supported by provider")
)

func Default() []Provider {
	return []Provider{
		keyProvider("anthropic", "ANTHROPIC_API_KEY", func(key string, baseURL string) agentkit.Provider {
			if baseURL != "" {
				return anthropic.New(anthropic.APIKey(key), anthropic.WithBaseURL(baseURL))
			}
			return anthropic.New(anthropic.APIKey(key))
		}),
		keyProvider("google", "GEMINI_API_KEY", func(key string, baseURL string) agentkit.Provider {
			if baseURL != "" {
				return google.New(google.APIKey(key), google.WithBaseURL(baseURL))
			}
			return google.New(google.APIKey(key))
		}),
		openAIProvider(),
		keyProvider("openrouter", "OPENROUTER_API_KEY", func(key string, baseURL string) agentkit.Provider {
			if baseURL != "" {
				return openrouter.New(openrouter.APIKey(key), openrouter.WithBaseURL(baseURL))
			}
			return openrouter.New(openrouter.APIKey(key))
		}),
		keyProvider("zai", "ZAI_API_KEY", func(key string, baseURL string) agentkit.Provider {
			if baseURL != "" {
				return zai.New(zai.APIKey(key), zai.WithBaseURL(baseURL))
			}
			return zai.New(zai.APIKey(key))
		}),
	}
}

func keyProvider(name, envKey string, construct func(key, baseURL string) agentkit.Provider) Provider {
	p := Provider{Name: name, EnvKey: envKey, Methods: []AuthMethod{AuthKey}}
	p.New = func(getenv func(string) string, opts Options) (agentkit.Provider, error) {
		auth := resolvedAuth(p, opts.Auth)
		if !supports(p, auth) {
			return nil, fmt.Errorf("%w: provider %s method %s", ErrAuthUnsupported, p.Name, auth)
		}
		key := getenv(p.EnvKey)
		if key == "" {
			return nil, fmt.Errorf("%w: %s", ErrMissingKey, p.EnvKey)
		}
		return construct(key, opts.BaseURL), nil
	}
	return p
}

func openAIProvider() Provider {
	p := Provider{
		Name:    "openai",
		EnvKey:  "OPENAI_API_KEY",
		Methods: []AuthMethod{AuthSub, AuthKey},
	}
	p.New = func(getenv func(string) string, opts Options) (agentkit.Provider, error) {
		auth := resolvedAuth(p, opts.Auth)
		if !supports(p, auth) {
			return nil, fmt.Errorf("%w: provider %s method %s", ErrAuthUnsupported, p.Name, auth)
		}
		var credential openai.Credential
		switch auth {
		case AuthSub:
			store, err := subscription.Load(opts.AuthFile)
			if err != nil {
				return nil, fmt.Errorf("load subscription auth file %q: %w", opts.AuthFile, err)
			}
			credential = openai.Subscription(store)
		case AuthKey:
			key := getenv(p.EnvKey)
			if key == "" {
				return nil, fmt.Errorf("%w: %s", ErrMissingKey, p.EnvKey)
			}
			credential = openai.APIKey(key)
		}
		if opts.BaseURL != "" {
			return openai.New(credential, openai.WithBaseURL(opts.BaseURL)), nil
		}
		return openai.New(credential), nil
	}
	return p
}

func resolvedAuth(provider Provider, requested AuthMethod) AuthMethod {
	if requested != "" {
		return requested
	}
	return provider.Methods[0]
}

func supports(provider Provider, method AuthMethod) bool {
	for _, candidate := range provider.Methods {
		if candidate == method {
			return true
		}
	}
	return false
}

func Lookup(cat []Provider, name string) (Provider, bool) {
	for _, provider := range cat {
		if provider.Name == name {
			return provider, true
		}
	}
	return Provider{}, false
}

func Models(name string) []akcatalog.Entry {
	entries := akcatalog.ListByProvider(name)
	chat := entries[:0]
	for _, entry := range entries {
		if entry.Embedding == nil {
			chat = append(chat, entry)
		}
	}
	return chat
}

func Resolve(provider, model string) (routeProvider, wireModel string, entry akcatalog.Entry, ok bool) {
	return akcatalog.Resolve(provider, model)
}
