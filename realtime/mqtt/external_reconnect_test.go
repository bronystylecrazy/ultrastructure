package mqtt_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	mmqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
)

func TestExternalReconnectAndResubscribe(t *testing.T) {
	addr := reserveTCPAddr(t)

	server := startTestBroker(t, addr)
	defer func() { _ = server.Close() }()

	client, err := usmqtt.NewExternal(usmqtt.ExternalConfig{
		Endpoint:       addr,
		ClientID:       "external-reconnect-test",
		ConnectTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewExternal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = client.Stop(context.Background()) }()

	received := make(chan string, 16)
	if err := client.Subscribe("smoke/reconnect", 1, func(_ *mmqtt.Client, _ packets.Subscription, pk packets.Packet) {
		received <- string(pk.Payload)
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	publishUntilReceived(t, server, received, "smoke/reconnect", "before-restart", 5*time.Second)

	if err := server.Close(); err != nil {
		t.Fatalf("close first broker: %v", err)
	}

	server = startTestBroker(t, addr)

	// Reconnect backoff starts at 1s; give enough room for connect + resubscribe.
	publishUntilReceived(t, server, received, "smoke/reconnect", "after-restart", 12*time.Second)
}

func TestExternalWSSReconnectAndResubscribe(t *testing.T) {
	addr := reserveTCPAddr(t)

	serverTLS := mustServerTLSConfig(t)
	server := startTestWSSBroker(t, addr, serverTLS)
	defer func() { _ = server.Close() }()

	client, err := usmqtt.NewExternal(usmqtt.ExternalConfig{
		Endpoint:       "wss://" + addr,
		ClientID:       "external-wss-reconnect-test",
		ConnectTimeout: 3 * time.Second,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true, // self-signed test cert
		},
	})
	if err != nil {
		t.Fatalf("NewExternal: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = client.Stop(context.Background()) }()

	received := make(chan string, 16)
	if err := client.Subscribe("smoke/wss/reconnect", 1, func(_ *mmqtt.Client, _ packets.Subscription, pk packets.Packet) {
		received <- string(pk.Payload)
	}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	publishUntilReceived(t, server, received, "smoke/wss/reconnect", "before-restart", 5*time.Second)

	if err := server.Close(); err != nil {
		t.Fatalf("close first broker: %v", err)
	}

	server = startTestWSSBroker(t, addr, serverTLS)

	publishUntilReceived(t, server, received, "smoke/wss/reconnect", "after-restart", 12*time.Second)
}

func startTestBroker(t *testing.T, addr string) *mmqtt.Server {
	t.Helper()

	server := mmqtt.New(&mmqtt.Options{InlineClient: true})
	if err := server.AddHook(new(auth.AllowHook), nil); err != nil {
		t.Fatalf("AddHook: %v", err)
	}
	if err := server.AddListener(listeners.NewTCP(listeners.Config{
		ID:      "t1",
		Address: addr,
	})); err != nil {
		t.Fatalf("AddListener: %v", err)
	}

	go func() {
		_ = server.Serve()
	}()

	waitUntil(t, 3*time.Second, 50*time.Millisecond, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, "broker did not start listening in time")

	return server
}

func startTestWSSBroker(t *testing.T, addr string, tlsCfg *tls.Config) *mmqtt.Server {
	t.Helper()

	server := mmqtt.New(&mmqtt.Options{InlineClient: true})
	if err := server.AddHook(new(auth.AllowHook), nil); err != nil {
		t.Fatalf("AddHook: %v", err)
	}
	if err := server.AddListener(listeners.NewWebsocket(listeners.Config{
		ID:        "ws1",
		Address:   addr,
		TLSConfig: tlsCfg,
	})); err != nil {
		t.Fatalf("AddListener: %v", err)
	}

	go func() {
		_ = server.Serve()
	}()

	waitUntil(t, 3*time.Second, 50*time.Millisecond, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, "wss broker did not start listening in time")

	return server
}

func mustServerTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		t.Fatalf("serial rand: %v", err)
	}

	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyPair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		MinVersion:   tls.VersionTLS12,
	}
}

func publishUntilReceived(t *testing.T, server *mmqtt.Server, received <-chan string, topic string, payload string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if err := server.Publish(topic, []byte(payload), false, 0); err != nil {
			// During restart windows, publish can fail transiently.
		}

		select {
		case got := <-received:
			if got != payload {
				t.Fatalf("payload mismatch: got=%q want=%q", got, payload)
			}
			return
		case <-time.After(200 * time.Millisecond):
		}

		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for payload %q", payload)
		}
	}
}

func reserveTCPAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve listen addr: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close reserved listener: %v", err)
	}
	return addr
}

func waitUntil(t *testing.T, timeout time.Duration, step time.Duration, check func() bool, failMsg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(step)
	}
	t.Fatal(failMsg)
}
