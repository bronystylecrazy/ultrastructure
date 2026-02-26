//go:build darwin

package license

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var ioRegPlatformUUIDRe = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

type platformHardwareDetector struct{}

func (platformHardwareDetector) Detect(ctx context.Context) (*HardwareBinding, error) {
	uuid, err := macOSPlatformUUID(ctx)
	if err != nil {
		return nil, err
	}

	return newOSKeystoreBinding(uuid), nil
}

func macOSPlatformUUID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	out, err := exec.CommandContext(ctx, "ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err == nil {
		if id, ok := parseIORegPlatformUUID(string(out)); ok {
			return id, nil
		}
	}

	alt, altErr := exec.CommandContext(ctx, "sysctl", "-n", "kern.uuid").Output()
	if altErr == nil {
		id := strings.TrimSpace(string(alt))
		if id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("%w: unable to read macOS platform UUID", ErrHardwareBindingUnavailable)
}

func parseIORegPlatformUUID(raw string) (string, bool) {
	m := ioRegPlatformUUIDRe.FindStringSubmatch(raw)
	if len(m) != 2 {
		return "", false
	}
	id := strings.TrimSpace(m[1])
	if id == "" {
		return "", false
	}
	return id, true
}
