package web

import (
	"strings"
	"testing"
)

func TestBuildFiberListenConfigTLSValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "no tls fields",
			cfg:  Config{},
		},
		{
			name: "cert only",
			cfg: Config{
				TLS: TLSConfig{CertFile: "server.crt"},
			},
			wantErr: "both cert_file and cert_key_file are required",
		},
		{
			name: "key only",
			cfg: Config{
				TLS: TLSConfig{CertKeyFile: "server.key"},
			},
			wantErr: "both cert_file and cert_key_file are required",
		},
		{
			name: "client ca only",
			cfg: Config{
				TLS: TLSConfig{CertClientFile: "ca.pem"},
			},
			wantErr: "tls client CA requires server cert_file and cert_key_file",
		},
		{
			name: "full tls config",
			cfg: Config{
				TLS: TLSConfig{
					CertFile:       "server.crt",
					CertKeyFile:    "server.key",
					CertClientFile: "ca.pem",
					TLSMinVersion:  "1.3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFiberListenConfig(tt.cfg)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error mismatch: got=%q want~=%q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildFiberListenConfig: %v", err)
			}
			if tt.cfg.TLS.CertFile != "" && got.CertFile != tt.cfg.TLS.CertFile {
				t.Fatalf("cert file mismatch: got=%q want=%q", got.CertFile, tt.cfg.TLS.CertFile)
			}
			if tt.cfg.TLS.CertKeyFile != "" && got.CertKeyFile != tt.cfg.TLS.CertKeyFile {
				t.Fatalf("cert key mismatch: got=%q want=%q", got.CertKeyFile, tt.cfg.TLS.CertKeyFile)
			}
		})
	}
}

