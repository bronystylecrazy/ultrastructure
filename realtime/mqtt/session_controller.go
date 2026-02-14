package mqtt

import (
	"context"
	"errors"
)

var ErrSessionControlUnsupported = errors.New("realtime/mqtt: session control is unsupported for this broker")

type SessionController interface {
	DisconnectClient(ctx context.Context, clientID string, reason string) error
}
