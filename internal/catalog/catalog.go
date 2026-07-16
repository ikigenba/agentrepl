package catalog

import (
	"errors"
	"fmt"

	"github.com/ikigenba/agentkit"
	"github.com/ikigenba/agentkit/anthropic"
	"github.com/ikigenba/agentkit/google"
	"github.com/ikigenba/agentkit/openai"
	"github.com/ikigenba/agentkit/zai"
)

type ProviderFunc func(apiKey string, opts Options) agentkit.Provider

type Options struct {
	BaseURL string
}

type Provider struct {
	Name      string
	EnvKey    string
	Models    []string
	New       ProviderFunc
	Reasoning agentkit.ReasoningInspector
}

var (
	ErrUnknownProvider = errors.New("unknown provider")
	ErrUnknownModel    = errors.New("unknown model for provider")
	ErrMissingKey      = errors.New("missing API key")
)

func Default() []Provider {
	return []Provider{
		{
			Name:   "anthropic",
			EnvKey: "ANTHROPIC_API_KEY",
			Models: []string{
				anthropic.ModelFable5,
				anthropic.ModelSonnet5,
				anthropic.ModelOpus48,
				anthropic.ModelSonnet46,
				anthropic.ModelHaiku45,
			},
			New: func(apiKey string, _ Options) agentkit.Provider {
				return anthropic.New(apiKey)
			},
			Reasoning: anthropic.Reasoning,
		},
		{
			Name:   "google",
			EnvKey: "GEMINI_API_KEY",
			Models: []string{
				google.ModelFlash25,
				google.ModelPro25,
				google.ModelFlash35,
				google.ModelLite31,
				google.ModelPro31Preview,
			},
			New: func(apiKey string, _ Options) agentkit.Provider {
				return google.New(apiKey)
			},
			Reasoning: google.Reasoning,
		},
		{
			Name:   "openai",
			EnvKey: "OPENAI_API_KEY",
			Models: []string{
				openai.ModelGPT56Luna,
				openai.ModelGPT56Sol,
				openai.ModelGPT56Terra,
				openai.ModelGPT55Pro,
				openai.ModelGPT55,
				openai.ModelGPT54,
				openai.ModelGPT54Mini,
				openai.ModelGPT54Nano,
			},
			New: func(apiKey string, _ Options) agentkit.Provider {
				return openai.New(apiKey)
			},
			Reasoning: openai.Reasoning,
		},
		{
			Name:   "zai",
			EnvKey: "ZAI_API_KEY",
			Models: []string{
				zai.ModelGLM52,
				zai.ModelGLM51,
				zai.ModelGLM47,
				zai.ModelGLM46,
			},
			New: func(apiKey string, opts Options) agentkit.Provider {
				if opts.BaseURL != "" {
					return zai.New(apiKey, zai.WithBaseURL(opts.BaseURL))
				}
				return zai.New(apiKey)
			},
			Reasoning: zai.Reasoning,
		},
	}
}

func Lookup(cat []Provider, name string) (Provider, bool) {
	for _, p := range cat {
		if p.Name == name {
			return p, true
		}
	}
	return Provider{}, false
}

func (p Provider) HasModel(model string) bool {
	for _, candidate := range p.Models {
		if candidate == model {
			return true
		}
	}
	return false
}

func (p Provider) Build(getenv func(string) string, opts Options) (agentkit.Provider, error) {
	apiKey := getenv(p.EnvKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%w: %s", ErrMissingKey, p.EnvKey)
	}
	return p.New(apiKey, opts), nil
}
