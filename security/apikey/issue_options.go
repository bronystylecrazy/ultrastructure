package apikey

import "time"

type IssueOption func(*IssueConfig)

type IssueConfig struct {
	Prefix    string
	Scopes    []string
	Metadata  map[string]string
	ExpiresAt *time.Time
}

func WithPrefix(prefix string) IssueOption {
	return func(c *IssueConfig) {
		c.Prefix = prefix
	}
}

func WithScopes(scopes ...string) IssueOption {
	return func(c *IssueConfig) {
		c.Scopes = append([]string(nil), scopes...)
	}
}

func WithMetadata(metadata map[string]string) IssueOption {
	return func(c *IssueConfig) {
		c.Metadata = cloneMap(metadata)
	}
}

func WithExpiresAt(expiresAt *time.Time) IssueOption {
	return func(c *IssueConfig) {
		c.ExpiresAt = expiresAt
	}
}

func resolveIssueConfig(opts ...IssueOption) IssueConfig {
	cfg := IssueConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}
