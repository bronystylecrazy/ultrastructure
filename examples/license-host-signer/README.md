# Host Signer CLI

This CLI runs on the host machine and signs SHA-256 digests for license challenge checks.
It is designed for offline, long-term deployments where the container app proves host key possession.

## Build

```bash
go build -o host-signer ./examples/license-host-signer
```

## Commands

### 1) Generate host key (one time)

```bash
./host-signer init-key --private-key /var/lib/host-signer/host-key.pem
```

Output includes:
- `public_key_der_b64`
- `pub_hash` (usable as license `device_binding.pub_hash`)

### 2) Print host identity (for license issuance)

```bash
./host-signer print-identity --private-key /var/lib/host-signer/host-key.pem
```

Example output:

```json
{
  "platform": "linux",
  "method": "host-key",
  "pub_hash": "..."
}
```

### 3) Run signer service

TCP:

```bash
./host-signer serve \
  --private-key /var/lib/host-signer/host-key.pem \
  --listen 127.0.0.1:8787 \
  --auth-token "<shared-token>"
```

Unix socket (Linux/macOS):

```bash
./host-signer serve \
  --private-key /var/lib/host-signer/host-key.pem \
  --unix-socket /var/run/host-signer.sock \
  --auth-token "<shared-token>"
```

Endpoints:
- `GET /v1/public-key`
- `POST /v1/sign` body: `{"digest_b64":"<base64url_sha256_digest>"}`

## Container Wiring

Linux host -> Linux container with Unix socket:

```bash
docker run --rm \
  -v /var/run/host-signer.sock:/run/host-signer.sock \
  -e HOST_SIGNER_URL=http://unix \
  -e HOST_SIGNER_SOCKET=/run/host-signer.sock \
  -e HOST_SIGNER_TOKEN="<shared-token>" \
  your-app:latest
```

Docker Desktop (macOS/Windows host -> Linux container): use TCP + `host.docker.internal`.

```bash
docker run --rm \
  -e HOST_SIGNER_URL=http://host.docker.internal:8787 \
  -e HOST_SIGNER_TOKEN="<shared-token>" \
  your-app:latest
```

## Security Notes

- Keep private key host-only, file mode `0600`, root-owned.
- Prefer hardware-backed key storage when possible (TPM/Secure Enclave/CNG).
- Protect the signer endpoint with auth token or mTLS.
- Keep signer bound to local host interfaces unless intentionally exposed.
