package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

type hostIdentity struct {
	Platform string `json:"platform"`
	Method   string `json:"method"`
	PubHash  string `json:"pub_hash"`
}

type publicKeyResponse struct {
	Alg             string `json:"alg"`
	PublicKeyDERB64 string `json:"public_key_der_b64"`
	PubHash         string `json:"pub_hash"`
}

type signRequest struct {
	DigestB64 string `json:"digest_b64"`
}

type signResponse struct {
	Alg          string `json:"alg"`
	SignatureB64 string `json:"signature_b64"`
}

func main() {
	root := &cobra.Command{
		Use:   "host-signer",
		Short: "Host signer for offline license hardware binding",
	}
	root.AddCommand(newInitKeyCommand())
	root.AddCommand(newPrintPublicKeyCommand())
	root.AddCommand(newPrintIdentityCommand())
	root.AddCommand(newSignCommand())
	root.AddCommand(newServeCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newInitKeyCommand() *cobra.Command {
	var privateKeyPath string

	cmd := &cobra.Command{
		Use:   "init-key",
		Short: "Generate a host ECDSA P-256 private key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(privateKeyPath) == "" {
				return errors.New("missing --private-key")
			}

			if err := os.MkdirAll(filepath.Dir(privateKeyPath), 0o755); err != nil {
				return fmt.Errorf("create private-key directory: %w", err)
			}
			if _, err := os.Stat(privateKeyPath); err == nil {
				return fmt.Errorf("private key already exists: %s", privateKeyPath)
			}

			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return fmt.Errorf("generate key: %w", err)
			}

			der, err := x509.MarshalECPrivateKey(key)
			if err != nil {
				return fmt.Errorf("marshal private key: %w", err)
			}
			block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: der}
			pemBytes := pem.EncodeToMemory(block)
			if err := os.WriteFile(privateKeyPath, pemBytes, 0o600); err != nil {
				return fmt.Errorf("write private key: %w", err)
			}

			pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return fmt.Errorf("marshal public key: %w", err)
			}

			out := publicKeyResponse{
				Alg:             "ecdsa-p256-sha256",
				PublicKeyDERB64: base64.RawURLEncoding.EncodeToString(pubDER),
				PubHash:         hashBytesToPubHash(pubDER),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	cmd.Flags().StringVar(&privateKeyPath, "private-key", "", "path to write PEM private key (0600)")
	return cmd
}

func newPrintPublicKeyCommand() *cobra.Command {
	var privateKeyPath string

	cmd := &cobra.Command{
		Use:   "print-public-key",
		Short: "Print public key and hash from existing private key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := readPrivateKey(privateKeyPath)
			if err != nil {
				return err
			}

			pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return fmt.Errorf("marshal public key: %w", err)
			}

			out := publicKeyResponse{
				Alg:             "ecdsa-p256-sha256",
				PublicKeyDERB64: base64.RawURLEncoding.EncodeToString(pubDER),
				PubHash:         hashBytesToPubHash(pubDER),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	cmd.Flags().StringVar(&privateKeyPath, "private-key", "", "path to PEM private key")
	_ = cmd.MarkFlagRequired("private-key")
	return cmd
}

func newPrintIdentityCommand() *cobra.Command {
	var privateKeyPath string

	cmd := &cobra.Command{
		Use:   "print-identity",
		Short: "Print license DeviceBinding-style identity from host key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := readPrivateKey(privateKeyPath)
			if err != nil {
				return err
			}

			pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return fmt.Errorf("marshal public key: %w", err)
			}

			out := hostIdentity{
				Platform: normalizedPlatform(),
				Method:   "host-key",
				PubHash:  hashBytesToPubHash(pubDER),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	cmd.Flags().StringVar(&privateKeyPath, "private-key", "", "path to PEM private key")
	_ = cmd.MarkFlagRequired("private-key")
	return cmd
}

func newSignCommand() *cobra.Command {
	var privateKeyPath string
	var digestB64 string

	cmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign a base64url SHA-256 digest",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := readPrivateKey(privateKeyPath)
			if err != nil {
				return err
			}

			digest, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(digestB64))
			if err != nil {
				return fmt.Errorf("decode digest: %w", err)
			}
			if len(digest) != sha256.Size {
				return fmt.Errorf("digest must be %d bytes, got %d", sha256.Size, len(digest))
			}

			sig, err := ecdsa.SignASN1(rand.Reader, key, digest)
			if err != nil {
				return fmt.Errorf("sign digest: %w", err)
			}
			out := signResponse{
				Alg:          "ecdsa-p256-sha256",
				SignatureB64: base64.RawURLEncoding.EncodeToString(sig),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}
	cmd.Flags().StringVar(&privateKeyPath, "private-key", "", "path to PEM private key")
	cmd.Flags().StringVar(&digestB64, "digest", "", "base64url SHA-256 digest")
	_ = cmd.MarkFlagRequired("private-key")
	_ = cmd.MarkFlagRequired("digest")
	return cmd
}

func newServeCommand() *cobra.Command {
	var privateKeyPath string
	var listenAddr string
	var unixSocketPath string
	var authToken string
	var readTimeout time.Duration
	var writeTimeout time.Duration

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run host signer HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := readPrivateKey(privateKeyPath)
			if err != nil {
				return err
			}
			pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return fmt.Errorf("marshal public key: %w", err)
			}
			pubHash := hashBytesToPubHash(pubDER)

			mux := http.NewServeMux()
			mux.HandleFunc("/v1/public-key", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
				if !authorized(r, authToken) {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
				writeJSON(w, http.StatusOK, publicKeyResponse{
					Alg:             "ecdsa-p256-sha256",
					PublicKeyDERB64: base64.RawURLEncoding.EncodeToString(pubDER),
					PubHash:         pubHash,
				})
			})
			mux.HandleFunc("/v1/sign", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
				if !authorized(r, authToken) {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}

				var req signRequest
				body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, "read body failed")
					return
				}
				if err := json.Unmarshal(body, &req); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}

				digest, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(req.DigestB64))
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid digest encoding")
					return
				}
				if len(digest) != sha256.Size {
					writeJSONError(w, http.StatusBadRequest, "digest must be sha256 (32 bytes)")
					return
				}

				sig, err := ecdsa.SignASN1(rand.Reader, key, digest)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, "sign failed")
					return
				}
				writeJSON(w, http.StatusOK, signResponse{
					Alg:          "ecdsa-p256-sha256",
					SignatureB64: base64.RawURLEncoding.EncodeToString(sig),
				})
			})

			server := &http.Server{
				Handler:      mux,
				ReadTimeout:  readTimeout,
				WriteTimeout: writeTimeout,
			}

			var listener net.Listener
			if strings.TrimSpace(unixSocketPath) != "" {
				if runtime.GOOS == "windows" {
					return errors.New("--unix-socket is not supported on windows")
				}
				if err := os.MkdirAll(filepath.Dir(unixSocketPath), 0o755); err != nil {
					return fmt.Errorf("create socket directory: %w", err)
				}
				_ = os.Remove(unixSocketPath)
				ln, err := net.Listen("unix", unixSocketPath)
				if err != nil {
					return fmt.Errorf("listen unix socket: %w", err)
				}
				if err := os.Chmod(unixSocketPath, 0o660); err != nil {
					_ = ln.Close()
					return fmt.Errorf("chmod unix socket: %w", err)
				}
				listener = ln
			} else {
				ln, err := net.Listen("tcp", listenAddr)
				if err != nil {
					return fmt.Errorf("listen tcp: %w", err)
				}
				listener = ln
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Serve(listener)
			}()

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
				return nil
			case err := <-errCh:
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
		},
	}

	cmd.Flags().StringVar(&privateKeyPath, "private-key", "", "path to PEM private key")
	cmd.Flags().StringVar(&listenAddr, "listen", "127.0.0.1:8787", "TCP listen address")
	cmd.Flags().StringVar(&unixSocketPath, "unix-socket", "", "Unix socket path (preferred on Linux/macOS)")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "optional bearer token; if set, requests need Authorization: Bearer <token>")
	cmd.Flags().DurationVar(&readTimeout, "read-timeout", 5*time.Second, "HTTP read timeout")
	cmd.Flags().DurationVar(&writeTimeout, "write-timeout", 5*time.Second, "HTTP write timeout")
	_ = cmd.MarkFlagRequired("private-key")
	return cmd
}

func readPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("missing --private-key")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("invalid PEM private key")
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key must be ECDSA, got %T", parsed)
	}
	return key, nil
}

func hashBytesToPubHash(value []byte) string {
	sum := sha256.Sum256(value)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func normalizedPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	const prefix = "Bearer "
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(got, prefix) {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(got, prefix)) == token
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
