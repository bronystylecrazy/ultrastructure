package license

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// PublicKeysB64 can be set only via go build -ldflags -X:
// -X 'github.com/bronystylecrazy/ultrastructure/security/license.PublicKeysB64=<base64url-json>'
//
// Decoded JSON shape:
// {"kid":"base64url-ed25519-public-key"}
var PublicKeysB64 string

func init() {
	keys, ok, err := loadPublicKeysFromLdflags(PublicKeysB64)
	if err != nil {
		panic(fmt.Sprintf("license: %v", err))
	}
	if ok {
		PublicKeys = keys
	}
}

func loadPublicKeysFromLdflags(rawB64 string) (map[string]string, bool, error) {
	rawB64 = strings.TrimSpace(rawB64)
	if rawB64 == "" {
		return nil, false, nil
	}

	decoded, err := decodeBase64Flexible(rawB64)
	if err != nil {
		return nil, false, fmt.Errorf("PublicKeysB64: decode base64 payload: %w", err)
	}
	payload := strings.TrimSpace(string(decoded))

	var keys map[string]string
	if err := json.Unmarshal([]byte(payload), &keys); err != nil {
		return nil, false, fmt.Errorf("PublicKeysB64: parse JSON map[kid]public_key: %w", err)
	}
	if len(keys) == 0 {
		return nil, false, fmt.Errorf("public keys ldflags payload is empty")
	}

	out := make(map[string]string, len(keys))
	for kid, pub := range keys {
		k := strings.TrimSpace(kid)
		v := strings.TrimSpace(pub)
		if k == "" {
			return nil, false, fmt.Errorf("public keys ldflags payload is invalid: empty kid")
		}
		if v == "" {
			return nil, false, fmt.Errorf("public keys ldflags payload is invalid: empty public key for kid %q", k)
		}
		out[k] = v
	}
	return out, true, nil
}

func decodeBase64Flexible(value string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(value); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(value)
}
