package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	getHardwareIDFullMethod    = "/ultrastructure.license.v1.HardwareIDService/GetHardwareID"
	getHardwareProofFullMethod = "/ultrastructure.license.v1.HardwareIDService/GetHardwareProof"
)

func main() {
	addr := flag.String("addr", "localhost:9090", "server address")
	caCert := flag.String("ca-cert", "", "server CA PEM path")
	serverName := flag.String("server-name", "", "TLS server name")
	clientCert := flag.String("client-cert", "", "optional client cert PEM path")
	clientKey := flag.String("client-key", "", "optional client key PEM path")
	authToken := flag.String("auth-token", "", "optional bearer token")
	expectedSignerPub := flag.String("expected-signer-pub", "", "expected proof signer public key b64 (recommended)")
	maxSkew := flag.Duration("max-skew", 30*time.Second, "max proof timestamp skew")
	flag.Parse()

	if strings.TrimSpace(*caCert) == "" {
		fatalf("--ca-cert is required")
	}

	tlsCfg, err := loadTLSConfig(*caCert, *serverName, *clientCert, *clientKey)
	if err != nil {
		fatalf("tls config: %v", err)
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	if err != nil {
		fatalf("dial: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if strings.TrimSpace(*authToken) != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+strings.TrimSpace(*authToken)))
	}

	var hw structpb.Struct
	if err := conn.Invoke(ctx, getHardwareIDFullMethod, &emptypb.Empty{}, &hw); err != nil {
		fatalf("GetHardwareID failed: %v", err)
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		fatalf("nonce random: %v", err)
	}
	nonceB64 := base64.RawURLEncoding.EncodeToString(nonce)

	proofReq, _ := structpb.NewStruct(map[string]any{"nonce_b64": nonceB64})
	var proof structpb.Struct
	if err := conn.Invoke(ctx, getHardwareProofFullMethod, proofReq, &proof); err != nil {
		fatalf("GetHardwareProof failed: %v", err)
	}

	if err := verifyProof(nonce, &proof, strings.TrimSpace(*expectedSignerPub), *maxSkew); err != nil {
		fatalf("proof verification failed: %v", err)
	}

	fmt.Printf("hardware id verified: platform=%s method=%s pub_hash=%s\n",
		mustString(&proof, "platform"),
		mustString(&proof, "method"),
		mustString(&proof, "pub_hash"),
	)
}

func verifyProof(nonce []byte, proof *structpb.Struct, expectedSignerPub string, maxSkew time.Duration) error {
	platform := mustString(proof, "platform")
	method := mustString(proof, "method")
	pubHash := mustString(proof, "pub_hash")
	nonceB64 := mustString(proof, "nonce_b64")
	tsStr := strconv.FormatInt(int64(mustNumber(proof, "ts_unix")), 10)
	sigB64 := mustString(proof, "signature_b64")
	signerPubB64 := mustString(proof, "signer_public_key_b64")

	if nonceB64 != base64.RawURLEncoding.EncodeToString(nonce) {
		return errors.New("nonce mismatch")
	}

	tsUnix := int64(mustNumber(proof, "ts_unix"))
	now := time.Now().UTC().Unix()
	if abs64(now-tsUnix) > int64(maxSkew.Seconds()) {
		return fmt.Errorf("proof timestamp skew too large: now=%d ts=%d", now, tsUnix)
	}

	if expectedSignerPub != "" && signerPubB64 != expectedSignerPub {
		return errors.New("unexpected signer public key")
	}

	pub, err := base64.RawURLEncoding.DecodeString(signerPubB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid signer public key")
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature")
	}
	msg := strings.Join([]string{
		"v1",
		nonceB64,
		platform,
		method,
		pubHash,
		tsStr,
	}, "|")
	if !ed25519.Verify(ed25519.PublicKey(pub), []byte(msg), sig) {
		return errors.New("bad proof signature")
	}
	return nil
}

func loadTLSConfig(caCert, serverName, clientCert, clientKey string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("read ca cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("parse ca cert")
	}

	cfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	if strings.TrimSpace(serverName) != "" {
		cfg.ServerName = strings.TrimSpace(serverName)
	}

	if strings.TrimSpace(clientCert) != "" || strings.TrimSpace(clientKey) != "" {
		if strings.TrimSpace(clientCert) == "" || strings.TrimSpace(clientKey) == "" {
			return nil, errors.New("both --client-cert and --client-key are required")
		}
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert/key: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func mustString(v *structpb.Struct, key string) string {
	if v == nil || v.Fields == nil || v.Fields[key] == nil {
		fatalf("missing response field %q", key)
	}
	out := strings.TrimSpace(v.Fields[key].GetStringValue())
	if out == "" {
		fatalf("empty response field %q", key)
	}
	return out
}

func mustNumber(v *structpb.Struct, key string) float64 {
	if v == nil || v.Fields == nil || v.Fields[key] == nil {
		fatalf("missing response field %q", key)
	}
	return v.Fields[key].GetNumberValue()
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
