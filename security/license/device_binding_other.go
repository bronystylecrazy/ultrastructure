//go:build !darwin && !linux && !windows

package license

import (
	"context"
	"fmt"
	"runtime"
)

func expectedDeviceBinding(ctx context.Context) (*DeviceBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: unsupported OS %s", ErrDeviceBindingUnavailable, runtime.GOOS)
}
