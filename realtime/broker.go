package realtime

import (
	"fmt"
	"regexp"
	"strings"

	us "github.com/bronystylecrazy/ultrastructure"
	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	"github.com/google/uuid"
)

const BrokerModeEmbedded = "embedded"
const BrokerModeExternal = "external"

func NewBroker(cfg Config) (usmqtt.Broker, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Broker.Mode))
	if mode == "" {
		mode = BrokerModeEmbedded
	}

	switch mode {
	case BrokerModeEmbedded:
		return usmqtt.NewServer()
	case BrokerModeExternal:
		tlsCfg, err := cfg.Broker.TLS.Load()
		if err != nil {
			return nil, fmt.Errorf("realtime: load broker tls config: %w", err)
		}

		clientID := strings.TrimSpace(cfg.Broker.ClientID)
		if clientID == "" {
			clientID = defaultExternalClientID()
		}

		return usmqtt.NewExternal(usmqtt.ExternalConfig{
			Endpoint:       cfg.Broker.Endpoint,
			ClientID:       clientID,
			Username:       cfg.Broker.Username,
			Password:       cfg.Broker.Password,
			CleanSession:   cfg.Broker.CleanSession,
			ConnectTimeout: cfg.Broker.ConnectTimeout,
			TLSConfig:      tlsCfg,
		})
	default:
		return nil, fmt.Errorf("realtime: invalid broker mode %q (allowed: %q, %q)", mode, BrokerModeEmbedded, BrokerModeExternal)
	}
}

var invalidClientIDRunes = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func defaultExternalClientID() string {
	base := strings.TrimSpace(us.Name)
	if base == "" {
		base = "ultrastructure"
	}
	base = invalidClientIDRunes.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "ultrastructure"
	}
	return base + "-" + uuid.NewString()
}
