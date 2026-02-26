# security/license

Minimal policy for offline, single-device licensing.

## Rules

1. Offline only: verify locally with embedded public keys.
2. One license = one device: every license must include one hardware binding.
3. Strict host lock: current host binding must exactly match license binding.

## Build-time key injection (`-X` only)

Inject verifier public keys at build time:

```bash
go build -ldflags "-X 'github.com/bronystylecrazy/ultrastructure/security/license.PublicKeysB64=${PUBLIC_KEYS_B64}'" ./...
```

`PUBLIC_KEYS_B64` must be base64/base64url JSON:

```json
{
  "kid-2026-01": "<base64url-ed25519-public-key>"
}
```

## Runtime usage

```go
payload, err := license.VerifyOfflineSingleDevice(ctx, token, time.Now().UTC())
```

Or explicit constructor:

```go
verifier, err := license.NewOfflineVerifier()
payload, err := verifier.VerifyForCurrentHost(ctx, token, time.Now().UTC())
```

## Rotation

Ship multiple `kid -> public_key` entries during transition:

1. Add new key (`kid-new`) to binaries.
2. Start signing with `kid-new`.
3. Keep old key until old licenses expire.
4. Remove old key in a later release.
