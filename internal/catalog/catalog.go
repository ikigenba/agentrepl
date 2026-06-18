package catalog

import "github.com/ikigenba/agentkit"

type ProviderFunc func(apiKey string) agentkit.Provider
