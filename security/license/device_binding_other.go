//go:build !darwin && !linux && !windows

package license

import (
	"context"
	"fmt"
	"runtime"
)

type platformHardwareDetector struct{}

func (platformHardwareDetector) Detect(ctx context.Context) (*HardwareBinding, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: unsupported OS %s", ErrHardwareBindingUnavailable, runtime.GOOS)
}
