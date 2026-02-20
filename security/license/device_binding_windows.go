//go:build windows

package license

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func expectedDeviceBinding(ctx context.Context) (*DeviceBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id, err := windowsMachineGUID()
	if err != nil {
		return nil, err
	}

	return &DeviceBinding{
		Platform: normalizedPlatform(),
		Method:   "os-keystore",
		PubHash:  hashToPubHash(strings.ToLower(id)),
	}, nil
}

func windowsMachineGUID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", fmt.Errorf("%w: open registry key: %v", ErrDeviceBindingUnavailable, err)
	}
	defer k.Close()

	v, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", fmt.Errorf("%w: read MachineGuid: %v", ErrDeviceBindingUnavailable, err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%w: empty MachineGuid", ErrDeviceBindingUnavailable)
	}
	return v, nil
}
