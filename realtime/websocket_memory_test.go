package realtime

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type heapSnapshot struct {
	alloc       uint64
	heapObjects uint64
}

func takeHeapSnapshot() heapSnapshot {
	runtime.GC()
	debug.FreeOSMemory()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return heapSnapshot{
		alloc:       ms.Alloc,
		heapObjects: ms.HeapObjects,
	}
}

func TestWebsocketHighPayloadRateNoRetainedGrowth(t *testing.T) {
	t.Parallel()

	const (
		payloadSize = 512 * 1024
		ratePerSec  = 15
		warmupMsgs  = 15
		testMsgs    = 60
	)

	totalMsgs := warmupMsgs + testMsgs
	payload := bytes.Repeat([]byte{0xAB}, payloadSize)
	interval := time.Second / time.Duration(ratePerSec)

	errCh := make(chan error, 1)
	doneCh := make(chan struct{})

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errCh <- fmt.Errorf("upgrade: %w", err)
			return
		}
		defer c.Close()

		conn := &wsConn{Conn: c.UnderlyingConn(), c: c}
		buf := make([]byte, payloadSize)
		for i := 0; i < totalMsgs; i++ {
			if _, err := io.ReadFull(conn, buf); err != nil {
				errCh <- fmt.Errorf("read message %d: %w", i, err)
				return
			}
		}

		close(doneCh)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer client.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for i := 0; i < warmupMsgs; i++ {
		<-ticker.C
		if err := client.WriteMessage(websocket.BinaryMessage, payload); err != nil {
			t.Fatalf("warmup write %d: %v", i, err)
		}
	}

	before := takeHeapSnapshot()

	for i := 0; i < testMsgs; i++ {
		<-ticker.C
		if err := client.WriteMessage(websocket.BinaryMessage, payload); err != nil {
			t.Fatalf("test write %d: %v", i, err)
		}
	}

	select {
	case err := <-errCh:
		t.Fatal(err)
	case <-doneCh:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for server to drain messages")
	}

	after := takeHeapSnapshot()

	allocGrowth := int64(after.alloc) - int64(before.alloc)
	objectsGrowth := int64(after.heapObjects) - int64(before.heapObjects)

	t.Logf(
		"heap retained bytes before=%d after=%d growth=%d; heap objects before=%d after=%d growth=%d",
		before.alloc,
		after.alloc,
		allocGrowth,
		before.heapObjects,
		after.heapObjects,
		objectsGrowth,
	)

	const maxRetainedGrowth = int64(12 * 1024 * 1024)
	if allocGrowth > maxRetainedGrowth {
		t.Fatalf("retained heap growth too high: got %d bytes, want <= %d", allocGrowth, maxRetainedGrowth)
	}
}
