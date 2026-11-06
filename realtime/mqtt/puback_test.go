package mqtt_test

import (
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/mochi-mqtt/server/v2/hooks/auth"

	usmqtt "github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
)

// TestServerPubackCarriesNoUserProperties: a QoS 1 PUBLISH with a user
// property gets the spec's 2-byte PUBACK (packet id only), not mochi's
// default echo of the user properties, which paho-mqtt (Python) and
// MQTT.js reject as malformed (mochi-mqtt/server#474).
func TestServerPubackCarriesNoUserProperties(t *testing.T) {
	srv, err := usmqtt.NewServer(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.AddHook(new(auth.AllowHook), nil); err != nil {
		t.Fatalf("AddHook: %v", err)
	}
	client, conn := net.Pipe()
	defer client.Close()
	errc := make(chan error, 1)
	go func() { errc <- srv.EstablishConnection("test", conn) }()
	defer func() {
		select {
		case err := <-errc:
			t.Logf("EstablishConnection: %v", err)
		default:
		}
	}()
	_ = client.SetDeadline(time.Now().Add(5 * time.Second))

	// CONNECT, MQTT 5, clean start, keepalive 60, no properties, client id "c".
	write(t, client, 0x10, 0x0e, 0x00, 0x04, 'M', 'Q', 'T', 'T', 0x05, 0x02, 0x00, 0x3c, 0x00, 0x00, 0x01, 'c')
	if typ, body := read(t, client); typ != 0x20 || len(body) < 2 || body[1] != 0x00 {
		t.Fatalf("CONNACK: type %#x body %x", typ, body)
	}

	// PUBLISH QoS 1 to "t", packet id 1, user property k=v, payload "x".
	write(t, client, 0x32, 0x0e, 0x00, 0x01, 't', 0x00, 0x01, 0x07, 0x26, 0x00, 0x01, 'k', 0x00, 0x01, 'v', 'x')
	typ, body := read(t, client)
	if typ != 0x40 {
		t.Fatalf("got packet type %#x, want PUBACK", typ)
	}
	if len(body) != 2 || body[0] != 0x00 || body[1] != 0x01 {
		t.Fatalf("PUBACK body %x, want 0001 (packet id only, no reason code or properties)", body)
	}
}

func write(t *testing.T, c net.Conn, b ...byte) {
	t.Helper()
	if _, err := c.Write(b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// read returns one packet's first byte and body; remaining lengths here
// all fit in one byte.
func read(t *testing.T, c net.Conn) (byte, []byte) {
	t.Helper()
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c, hdr); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if hdr[1]&0x80 != 0 {
		t.Fatalf("remaining length over 127 not expected: %x", hdr)
	}
	body := make([]byte, hdr[1])
	if _, err := io.ReadFull(c, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return hdr[0], body
}
