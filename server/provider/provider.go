package provider

import (
	"fmt"

	"cursortab/engine"
	"cursortab/provider/fim"
	"cursortab/provider/inline"
	"cursortab/provider/sweep"
	"cursortab/provider/zeta"
	"cursortab/types"
)

// NewProvider creates a new provider instance based on the type
func NewProvider(providerType types.ProviderType, config *types.ProviderConfig) (engine.Provider, error) {
	switch providerType {
	case types.ProviderTypeInline:
		return inline.NewProvider(config)
	case types.ProviderTypeFIM:
		return fim.NewProvider(config)
	case types.ProviderTypeSweep:
		return sweep.NewProvider(config)
	case types.ProviderTypeZeta:
		return zeta.NewProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
