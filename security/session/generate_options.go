package session

type GenerateOption func(*GenerateConfig)

type GenerateConfig struct {
	AccessClaims  map[string]any
	RefreshClaims map[string]any
}

func WithAccessClaims(claims map[string]any) GenerateOption {
	return func(c *GenerateConfig) {
		c.AccessClaims = cloneClaimsMap(claims)
	}
}

func WithRefreshClaims(claims map[string]any) GenerateOption {
	return func(c *GenerateConfig) {
		c.RefreshClaims = cloneClaimsMap(claims)
	}
}

func resolveGenerateConfig(opts ...GenerateOption) GenerateConfig {
	cfg := GenerateConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func cloneClaimsMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
