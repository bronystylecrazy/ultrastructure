//go:build !windows

package otel

import (
	"errors"

	"go.uber.org/zap/zapcore"
)

func openDaemonFileCore(path string) (zapcore.Core, func() error, error) {
	return nil, nil, errors.New("daemon file logging is only supported on windows")
}
