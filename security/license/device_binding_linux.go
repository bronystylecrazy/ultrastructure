//go:build linux

package license

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type platformHardwareDetector struct{}

func (platformHardwareDetector) Detect(ctx context.Context) (*HardwareBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id, err := readLinuxHardwareID()
	if err != nil {
		return nil, err
	}

	return newOSKeystoreBinding(id), nil
}

func readLinuxHardwareID() (string, error) {
	for _, p := range []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
		"/sys/class/dmi/id/product_uuid",
		"/sys/devices/virtual/dmi/id/product_uuid",
	} {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(raw))
		if v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("%w: unable to read linux hardware id", ErrHardwareBindingUnavailable)
}
