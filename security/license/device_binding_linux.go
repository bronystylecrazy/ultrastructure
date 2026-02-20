//go:build linux

package license

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func expectedDeviceBinding(ctx context.Context) (*DeviceBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id, err := linuxMachineID()
	if err != nil {
		return nil, err
	}

	return &DeviceBinding{
		Platform: normalizedPlatform(),
		Method:   "os-keystore",
		PubHash:  hashToPubHash(strings.ToLower(id)),
	}, nil
}

func linuxMachineID() (string, error) {
	paths := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		id := strings.TrimSpace(string(raw))
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: unable to read linux machine-id", ErrDeviceBindingUnavailable)
}
