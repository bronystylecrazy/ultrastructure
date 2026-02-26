//go:build windows

package license

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type platformHardwareDetector struct{}

func (platformHardwareDetector) Detect(ctx context.Context) (*HardwareBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id, err := windowsMachineGUID()
	if err != nil {
		return nil, err
	}

	return newOSKeystoreBinding(id), nil
}

func windowsMachineGUID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", fmt.Errorf("%w: open registry key: %v", ErrHardwareBindingUnavailable, err)
	}
	defer k.Close()

	v, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", fmt.Errorf("%w: read MachineGuid: %v", ErrHardwareBindingUnavailable, err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%w: empty MachineGuid", ErrHardwareBindingUnavailable)
	}
	return v, nil
}
