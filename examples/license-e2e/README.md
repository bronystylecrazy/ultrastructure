# license-e2e

End-to-end offline single-device license flow using `security/license`.

## 1) Issue license for current host

```bash
go run ./examples/license-e2e issue --out /tmp/license.json --kid kid-demo-001
```

This prints:
- `PUBLIC_KEYS_JSON`
- `PUBLIC_KEYS_B64`
- a ready `go build -ldflags -X ...` command

## 2) Build verifier binary with embedded public keys

Use printed `PUBLIC_KEYS_B64`:

```bash
go build -ldflags "-X 'github.com/bronystylecrazy/ultrastructure/security/license.PublicKeysB64=<PUBLIC_KEYS_B64>'" -o ./license-e2e ./examples/license-e2e
```

## 3) Verify offline on device

```bash
./license-e2e verify --license /tmp/license.json
```

Verification enforces:
- valid Ed25519 signature
- time window (`expiry` unless `never_expires=true`)
- strict device binding match to current host
