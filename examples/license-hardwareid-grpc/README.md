# license-hardwareid-grpc

Hardened gRPC hardware-id service with TLS/mTLS, auth token, and signed nonce proof.

## Server methods

- `/ultrastructure.license.v1.HardwareIDService/GetHardwareID`
- `/ultrastructure.license.v1.HardwareIDService/GetHardwareProof`

`GetHardwareProof` signs a freshness challenge (`nonce_b64`) so clients can verify authenticity and freshness.

## 1) Generate local certs (dev only)

```bash
# CA
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout /tmp/ca.key -out /tmp/ca.crt -subj "/CN=dev-ca"

# Server cert
openssl req -newkey rsa:2048 -nodes -keyout /tmp/server.key -out /tmp/server.csr -subj "/CN=localhost"
openssl x509 -req -in /tmp/server.csr -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial -days 365 -out /tmp/server.crt

# Client cert (for mTLS)
openssl req -newkey rsa:2048 -nodes -keyout /tmp/client.key -out /tmp/client.csr -subj "/CN=client"
openssl x509 -req -in /tmp/client.csr -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial -days 365 -out /tmp/client.crt
```

## 2) Run server (TLS + mTLS + auth + proof signer)

```bash
go run ./examples/license-hardwareid-grpc \
  --addr :9090 \
  --tls-cert /tmp/server.crt \
  --tls-key /tmp/server.key \
  --tls-client-ca /tmp/ca.crt \
  --auth-token dev-token \
  --proof-seed-hex 000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f
```

Server prints `proof_signer_public_key_b64`; pin this in clients.

## 3) Run secure client

```bash
go run ./examples/license-hardwareid-grpc-client \
  --addr localhost:9090 \
  --ca-cert /tmp/ca.crt \
  --server-name localhost \
  --client-cert /tmp/client.crt \
  --client-key /tmp/client.key \
  --auth-token dev-token \
  --expected-signer-pub <proof_signer_public_key_b64>
```

The client verifies:
- TLS server identity (CA + server name)
- optional mTLS client auth
- auth token
- nonce echo
- proof timestamp skew
- Ed25519 signature over proof payload
