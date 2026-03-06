package main

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	licensepkg "github.com/bronystylecrazy/ultrastructure/security/license"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	hardwareIDServiceName      = "ultrastructure.license.v1.HardwareIDService"
	getHardwareIDMethodName    = "GetHardwareID"
	getHardwareProofMethodName = "GetHardwareProof"
	getHardwareIDFullMethod    = "/" + hardwareIDServiceName + "/" + getHardwareIDMethodName
	getHardwareProofFullMethod = "/" + hardwareIDServiceName + "/" + getHardwareProofMethodName
)

type proofSigner struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

type hardwareIDService struct {
	detector licensepkg.HardwareDetector
	signer   *proofSigner
}

func (s *hardwareIDService) GetHardwareID(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	binding, err := s.detectBinding(ctx)
	if err != nil {
		return nil, err
	}

	out, err := structpb.NewStruct(map[string]any{
		"platform": binding.Platform,
		"method":   binding.Method,
		"pub_hash": binding.PubHash,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode response: %v", err)
	}
	return out, nil
}

func (s *hardwareIDService) GetHardwareProof(ctx context.Context, req *structpb.Struct) (*structpb.Struct, error) {
	if s == nil || s.signer == nil || len(s.signer.private) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "proof signer is not configured")
	}
	nonceB64, err := getStringField(req, "nonce_b64")
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceB64)
	if err != nil || len(nonce) == 0 {
		return nil, status.Error(codes.InvalidArgument, "nonce_b64 must be base64url non-empty bytes")
	}
	if len(nonce) > 128 {
		return nil, status.Error(codes.InvalidArgument, "nonce too long")
	}

	binding, err := s.detectBinding(ctx)
	if err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Unix()
	message := proofMessage(nonce, binding, ts)
	sig := ed25519.Sign(s.signer.private, []byte(message))

	out, err := structpb.NewStruct(map[string]any{
		"platform":              binding.Platform,
		"method":                binding.Method,
		"pub_hash":              binding.PubHash,
		"nonce_b64":             nonceB64,
		"ts_unix":               float64(ts),
		"signature_b64":         base64.RawURLEncoding.EncodeToString(sig),
		"signer_public_key_b64": base64.RawURLEncoding.EncodeToString(s.signer.public),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode proof response: %v", err)
	}
	return out, nil
}

func (s *hardwareIDService) detectBinding(ctx context.Context) (*licensepkg.HardwareBinding, error) {
	if s == nil || s.detector == nil {
		return nil, status.Error(codes.Internal, "hardware detector is not configured")
	}
	binding, err := s.detector.Detect(ctx)
	if err != nil {
		if errors.Is(err, licensepkg.ErrHardwareBindingUnavailable) {
			return nil, status.Errorf(codes.Unavailable, "hardware id unavailable: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "detect hardware id: %v", err)
	}
	return binding, nil
}

func registerHardwareIDService(s *grpc.Server, impl *hardwareIDService) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: hardwareIDServiceName,
		HandlerType: (*hardwareIDService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: getHardwareIDMethodName,
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					in := new(emptypb.Empty)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*hardwareIDService).GetHardwareID(ctx, in)
					}
					info := &grpc.UnaryServerInfo{Server: srv, FullMethod: getHardwareIDFullMethod}
					handler := func(ctx context.Context, req any) (any, error) {
						return srv.(*hardwareIDService).GetHardwareID(ctx, req.(*emptypb.Empty))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
			{
				MethodName: getHardwareProofMethodName,
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					in := new(structpb.Struct)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*hardwareIDService).GetHardwareProof(ctx, in)
					}
					info := &grpc.UnaryServerInfo{Server: srv, FullMethod: getHardwareProofFullMethod}
					handler := func(ctx context.Context, req any) (any, error) {
						return srv.(*hardwareIDService).GetHardwareProof(ctx, req.(*structpb.Struct))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
	}, impl)
}

func authInterceptor(token string) grpc.UnaryServerInterceptor {
	token = strings.TrimSpace(token)
	if token == "" {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	expected := "Bearer " + token
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}
		if subtle.ConstantTimeCompare([]byte(vals[0]), []byte(expected)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}
		return handler(ctx, req)
	}
}

func loadTLSConfig(serverCert, serverKey, clientCA string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}
	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}
	if strings.TrimSpace(clientCA) != "" {
		caPEM, err := os.ReadFile(clientCA)
		if err != nil {
			return nil, fmt.Errorf("read client ca: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("parse client ca pem")
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}

func loadProofSigner(seedHex string) (*proofSigner, error) {
	seedHex = strings.TrimSpace(seedHex)
	if seedHex == "" {
		return nil, nil
	}
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		return nil, fmt.Errorf("invalid proof-seed-hex: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid proof seed length: got %d want %d", len(seed), ed25519.SeedSize)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return &proofSigner{private: priv, public: pub}, nil
}

func proofMessage(nonce []byte, binding *licensepkg.HardwareBinding, ts int64) string {
	return strings.Join([]string{
		"v1",
		base64.RawURLEncoding.EncodeToString(nonce),
		binding.Platform,
		binding.Method,
		binding.PubHash,
		strconv.FormatInt(ts, 10),
	}, "|")
}

func getStringField(v *structpb.Struct, key string) (string, error) {
	if v == nil || v.Fields == nil {
		return "", fmt.Errorf("missing field %q", key)
	}
	f, ok := v.Fields[key]
	if !ok || f == nil {
		return "", fmt.Errorf("missing field %q", key)
	}
	s := strings.TrimSpace(f.GetStringValue())
	if s == "" {
		return "", fmt.Errorf("empty field %q", key)
	}
	return s, nil
}

func main() {
	addr := flag.String("addr", ":9090", "gRPC listen address")
	enableReflection := flag.Bool("reflection", true, "enable gRPC reflection")
	serverCert := flag.String("tls-cert", "", "server TLS cert PEM path")
	serverKey := flag.String("tls-key", "", "server TLS key PEM path")
	clientCA := flag.String("tls-client-ca", "", "client CA PEM path (enables mTLS)")
	authToken := flag.String("auth-token", "", "optional bearer token for unary RPC authorization")
	proofSeedHex := flag.String("proof-seed-hex", "", "optional ed25519 seed hex for nonce-proof signing (32 bytes)")
	flag.Parse()

	if strings.TrimSpace(*serverCert) == "" || strings.TrimSpace(*serverKey) == "" {
		log.Fatal("--tls-cert and --tls-key are required")
	}

	tlsCfg, err := loadTLSConfig(*serverCert, *serverKey, *clientCA)
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}
	signer, err := loadProofSigner(*proofSeedHex)
	if err != nil {
		log.Fatalf("proof signer: %v", err)
	}
	if signer != nil {
		fmt.Printf("proof_signer_public_key_b64=%s\n", base64.RawURLEncoding.EncodeToString(signer.public))
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}

	srv := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.ChainUnaryInterceptor(authInterceptor(*authToken)),
	)
	registerHardwareIDService(srv, &hardwareIDService{detector: licensepkg.NewHardwareDetector(), signer: signer})
	if *enableReflection {
		reflection.Register(srv)
	}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		srv.GracefulStop()
	}()

	fmt.Printf("hardware-id grpc server listening on %s\n", *addr)
	fmt.Printf("methods:\n  %s\n  %s\n", getHardwareIDFullMethod, getHardwareProofFullMethod)
	if err := srv.Serve(ln); err != nil {
		log.Fatalf("serve grpc: %v", err)
	}
}
