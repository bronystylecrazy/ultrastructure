package mqtt

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

var ErrExternalBrokerEndpointRequired = errors.New("realtime/mqtt: external broker endpoint is required")
var ErrExternalBrokerNotConnected = errors.New("realtime/mqtt: external broker is not connected")

const reconnectBackoffMin = 1 * time.Second
const reconnectBackoffMax = 30 * time.Second

type ExternalConfig struct {
	Endpoint       string
	ClientID       string
	Username       string
	Password       string
	CleanSession   bool
	ConnectTimeout time.Duration
	TLSConfig      *tls.Config
}

type External struct {
	cfg      ExternalConfig
	packetID atomic.Uint32

	mu           sync.RWMutex
	conn         net.Conn
	reader       *bufio.Reader
	started      bool
	closing      bool
	reconnecting bool
	handlers     map[string]map[int]mqtt.InlineSubFn
	writeMu      sync.Mutex
}

func NewExternal(cfg ExternalConfig) (*External, error) {
	if cfg.Endpoint == "" {
		return nil, ErrExternalBrokerEndpointRequired
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "ultrastructure-realtime"
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}

	return &External{
		cfg:      cfg,
		handlers: make(map[string]map[int]mqtt.InlineSubFn),
	}, nil
}

func (e *External) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	e.closing = false
	e.mu.Unlock()

	conn, reader, err := e.establishConnection(ctx)
	if err != nil {
		return err
	}
	e.setConnected(conn, reader)

	go e.readLoop(conn, reader)

	filters := e.subscriptionFilters()
	for _, filter := range filters {
		if err := e.subscribeRemote(filter); err != nil {
			e.handleConnectionLoss(conn, false)
			return err
		}
	}

	return nil
}

func (e *External) Stop(context.Context) error {
	e.mu.Lock()
	e.closing = true
	conn := e.conn
	e.conn = nil
	e.reader = nil
	e.started = false
	e.mu.Unlock()

	if conn == nil {
		return nil
	}

	_ = e.writePacket(conn, packets.Packet{
		FixedHeader: packets.FixedHeader{Type: packets.Disconnect},
	})
	return conn.Close()
}

func (e *External) Publish(topic string, payload []byte, retain bool, qos byte) error {
	conn, err := e.connectedConn()
	if err != nil {
		return err
	}

	pk := packets.Packet{
		FixedHeader: packets.FixedHeader{
			Type:   packets.Publish,
			Qos:    qos,
			Retain: retain,
		},
		TopicName: topic,
		Payload:   payload,
	}
	if qos > 0 {
		pk.PacketID = e.nextPacketID()
	}
	return e.writePacket(conn, pk)
}

func (e *External) PublishJSON(topic string, payload any, retain bool, qos byte) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return e.Publish(topic, p, retain, qos)
}

func (e *External) PublishString(topic string, payload string, retain bool, qos byte) error {
	return e.Publish(topic, []byte(payload), retain, qos)
}

func (e *External) DisconnectClient(context.Context, string, string) error {
	return ErrSessionControlUnsupported
}

func (e *External) Subscribe(filter string, subscriptionID int, handler mqtt.InlineSubFn) error {
	if handler == nil {
		return errors.New("realtime/mqtt: subscribe handler is nil")
	}

	e.mu.Lock()
	m, ok := e.handlers[filter]
	if !ok {
		m = make(map[int]mqtt.InlineSubFn)
		e.handlers[filter] = m
	}
	firstRemote := len(m) == 0
	m[subscriptionID] = handler
	started := e.started
	e.mu.Unlock()

	if !firstRemote || !started {
		return nil
	}

	return e.subscribeRemote(filter)
}

func (e *External) Unsubscribe(filter string, subscriptionID int) error {
	e.mu.Lock()
	m, ok := e.handlers[filter]
	if !ok {
		e.mu.Unlock()
		return nil
	}
	delete(m, subscriptionID)
	shouldRemoteUnsub := len(m) == 0
	if shouldRemoteUnsub {
		delete(e.handlers, filter)
	}
	started := e.started
	e.mu.Unlock()

	if !shouldRemoteUnsub || !started {
		return nil
	}

	conn, err := e.connectedConn()
	if err != nil {
		return err
	}
	return e.writePacket(conn, packets.Packet{
		FixedHeader: packets.FixedHeader{Type: packets.Unsubscribe, Qos: 1},
		PacketID:    e.nextPacketID(),
		Filters:     packets.Subscriptions{{Filter: filter}},
	})
}

func (e *External) readLoop(conn net.Conn, reader *bufio.Reader) {
	for {
		pk, err := readPacket(reader)
		if err != nil {
			e.handleConnectionLoss(conn, true)
			return
		}

		switch pk.FixedHeader.Type {
		case packets.Publish:
			e.dispatch(pk)
		case packets.Pingreq:
			_ = e.writePacket(conn, packets.Packet{
				FixedHeader: packets.FixedHeader{Type: packets.Pingresp},
			})
		}
	}
}

func (e *External) establishConnection(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	conn, err := dialExternalConn(ctx, e.cfg.Endpoint, e.cfg.ConnectTimeout, e.cfg.TLSConfig)
	if err != nil {
		return nil, nil, err
	}

	reader := bufio.NewReader(conn)
	if err := e.writePacket(conn, packets.Packet{
		FixedHeader:     packets.FixedHeader{Type: packets.Connect},
		ProtocolVersion: 4,
		Connect: packets.ConnectParams{
			ProtocolName:     []byte{'M', 'Q', 'T', 'T'},
			Clean:            e.cfg.CleanSession,
			ClientIdentifier: e.cfg.ClientID,
			Keepalive:        30,
			UsernameFlag:     e.cfg.Username != "",
			PasswordFlag:     e.cfg.Password != "",
			Username:         []byte(e.cfg.Username),
			Password:         []byte(e.cfg.Password),
		},
	}); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	if err := conn.SetReadDeadline(time.Now().Add(e.cfg.ConnectTimeout)); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	connack, err := readPacket(reader)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if connack.FixedHeader.Type != packets.Connack {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("realtime/mqtt: expected connack, got packet type=%d", connack.FixedHeader.Type)
	}
	if connack.ReasonCode != 0 {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("realtime/mqtt: external broker rejected connection, reason_code=%d", connack.ReasonCode)
	}

	return conn, reader, nil
}

func (e *External) setConnected(conn net.Conn, reader *bufio.Reader) {
	e.mu.Lock()
	e.conn = conn
	e.reader = reader
	e.started = true
	e.mu.Unlock()
}

func (e *External) subscriptionFilters() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	filters := make([]string, 0, len(e.handlers))
	for filter := range e.handlers {
		filters = append(filters, filter)
	}
	return filters
}

func (e *External) handleConnectionLoss(conn net.Conn, reconnect bool) {
	e.mu.Lock()
	if conn != nil && e.conn != nil && e.conn != conn {
		e.mu.Unlock()
		return
	}

	e.conn = nil
	e.reader = nil
	e.started = false

	shouldReconnect := reconnect && !e.closing && !e.reconnecting
	if shouldReconnect {
		e.reconnecting = true
	}
	e.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	if shouldReconnect {
		go e.reconnectLoop()
	}
}

func (e *External) reconnectLoop() {
	defer func() {
		e.mu.Lock()
		e.reconnecting = false
		e.mu.Unlock()
	}()

	backoff := reconnectBackoffMin
	for {
		e.mu.RLock()
		closing := e.closing
		e.mu.RUnlock()
		if closing {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), e.cfg.ConnectTimeout)
		conn, reader, err := e.establishConnection(ctx)
		cancel()
		if err == nil {
			e.setConnected(conn, reader)

			filters := e.subscriptionFilters()
			var resubErr error
			for _, filter := range filters {
				if err := e.subscribeRemote(filter); err != nil {
					resubErr = err
					break
				}
			}

			if resubErr == nil {
				go e.readLoop(conn, reader)
				return
			}

			e.handleConnectionLoss(conn, false)
		}

		if !e.waitForReconnectBackoff(backoff) {
			return
		}
		backoff *= 2
		if backoff > reconnectBackoffMax {
			backoff = reconnectBackoffMax
		}
	}
}

func (e *External) waitForReconnectBackoff(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return true
		case <-ticker.C:
			e.mu.RLock()
			closing := e.closing
			e.mu.RUnlock()
			if closing {
				return false
			}
		}
	}
}

func (e *External) dispatch(pk packets.Packet) {
	e.mu.RLock()
	type target struct {
		filter  string
		handler mqtt.InlineSubFn
	}
	matched := make([]target, 0)
	for f, handlers := range e.handlers {
		if !topicMatchesFilter(pk.TopicName, f) {
			continue
		}
		for _, h := range handlers {
			matched = append(matched, target{filter: f, handler: h})
		}
	}
	e.mu.RUnlock()

	if len(matched) == 0 {
		return
	}

	cl := &mqtt.Client{
		ID: e.cfg.ClientID,
		Properties: mqtt.ClientProperties{
			Username: []byte(e.cfg.Username),
		},
	}
	for _, t := range matched {
		sub := packets.Subscription{Filter: t.filter}
		t.handler(cl, sub, pk.Copy(false))
	}
}

func (e *External) subscribeRemote(filter string) error {
	conn, err := e.connectedConn()
	if err != nil {
		return err
	}
	return e.writePacket(conn, packets.Packet{
		FixedHeader: packets.FixedHeader{Type: packets.Subscribe, Qos: 1},
		PacketID:    e.nextPacketID(),
		Filters: packets.Subscriptions{
			{
				Filter: filter,
				Qos:    QoS0,
			},
		},
	})
}

func (e *External) connectedConn() (net.Conn, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.started || e.conn == nil {
		return nil, ErrExternalBrokerNotConnected
	}
	return e.conn, nil
}

func (e *External) nextPacketID() uint16 {
	id := uint16(e.packetID.Add(1) % 65535)
	if id == 0 {
		id = uint16(e.packetID.Add(1) % 65535)
		if id == 0 {
			id = 1
		}
	}
	return id
}

func (e *External) writePacket(conn net.Conn, pk packets.Packet) error {
	pk.ProtocolVersion = 4
	buf := new(bytes.Buffer)
	var err error
	switch pk.FixedHeader.Type {
	case packets.Connect:
		err = pk.ConnectEncode(buf)
	case packets.Publish:
		err = pk.PublishEncode(buf)
	case packets.Subscribe:
		err = pk.SubscribeEncode(buf)
	case packets.Unsubscribe:
		err = pk.UnsubscribeEncode(buf)
	case packets.Pingreq:
		err = pk.PingreqEncode(buf)
	case packets.Pingresp:
		err = pk.PingrespEncode(buf)
	case packets.Disconnect:
		err = pk.DisconnectEncode(buf)
	default:
		err = fmt.Errorf("realtime/mqtt: unsupported outbound packet type=%d", pk.FixedHeader.Type)
	}
	if err != nil {
		return err
	}

	e.writeMu.Lock()
	defer e.writeMu.Unlock()
	if err := conn.SetWriteDeadline(time.Now().Add(e.cfg.ConnectTimeout)); err != nil {
		if pk.FixedHeader.Type != packets.Connect && pk.FixedHeader.Type != packets.Disconnect {
			e.handleConnectionLoss(conn, true)
		}
		return err
	}
	defer conn.SetWriteDeadline(time.Time{})
	_, err = conn.Write(buf.Bytes())
	if err != nil && pk.FixedHeader.Type != packets.Connect && pk.FixedHeader.Type != packets.Disconnect {
		e.handleConnectionLoss(conn, true)
	}
	return err
}

func readPacket(reader *bufio.Reader) (packets.Packet, error) {
	var pk packets.Packet
	headerByte, err := reader.ReadByte()
	if err != nil {
		return pk, err
	}
	fh := packets.FixedHeader{}
	if err := fh.Decode(headerByte); err != nil {
		return pk, err
	}
	remaining, _, err := packets.DecodeLength(reader)
	if err != nil {
		return pk, err
	}
	fh.Remaining = remaining

	data := make([]byte, remaining)
	if _, err := io.ReadFull(reader, data); err != nil {
		return pk, err
	}
	pk = packets.Packet{
		FixedHeader:     fh,
		ProtocolVersion: 4,
	}

	switch fh.Type {
	case packets.Connack:
		err = pk.ConnackDecode(data)
	case packets.Publish:
		err = pk.PublishDecode(data)
	case packets.Suback:
		err = pk.SubackDecode(data)
	case packets.Unsuback:
		err = pk.UnsubackDecode(data)
	case packets.Pingresp:
		err = pk.PingrespDecode(data)
	case packets.Pingreq:
		err = pk.PingreqDecode(data)
	default:
		// Unknown packet types are ignored by returning a packet with no decode error.
	}
	return pk, err
}

func normalizeMQTTEndpoint(endpoint string) string {
	e := strings.TrimSpace(endpoint)
	e = strings.TrimPrefix(e, "mqtt://")
	e = strings.TrimPrefix(e, "tcp://")
	e = strings.TrimPrefix(e, "mqtts://")
	e = strings.TrimPrefix(e, "ssl://")
	e = strings.TrimPrefix(e, "tls://")
	return e
}

func isTLSEndpoint(endpoint string) bool {
	e := strings.TrimSpace(endpoint)
	return strings.HasPrefix(e, "mqtts://") ||
		strings.HasPrefix(e, "ssl://") ||
		strings.HasPrefix(e, "tls://")
}

func dialExternalConn(ctx context.Context, endpoint string, timeout time.Duration, tlsCfg *tls.Config) (net.Conn, error) {
	e := strings.TrimSpace(endpoint)
	if e == "" {
		return nil, ErrExternalBrokerEndpointRequired
	}
	if strings.HasPrefix(e, "ws://") || strings.HasPrefix(e, "wss://") {
		dialer := websocket.Dialer{
			HandshakeTimeout: timeout,
			Subprotocols:     []string{"mqtt"},
			TLSClientConfig:  tlsCfg,
		}
		conn, _, err := dialer.DialContext(ctx, e, http.Header{})
		if err != nil {
			return nil, err
		}
		return &websocketConn{Conn: conn}, nil
	}

	addr := normalizeMQTTEndpoint(e)
	if isTLSEndpoint(e) {
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			Config:    tlsCfg,
		}
		return dialer.DialContext(ctx, "tcp", addr)
	}

	dialer := &net.Dialer{Timeout: timeout}
	return dialer.DialContext(ctx, "tcp", addr)
}

type websocketConn struct {
	Conn *websocket.Conn
	r    io.Reader
}

func (ws *websocketConn) Read(p []byte) (int, error) {
	if ws.r == nil {
		op, r, err := ws.Conn.NextReader()
		if err != nil {
			return 0, err
		}

		if op != websocket.BinaryMessage {
			return 0, fmt.Errorf("realtime/mqtt: websocket message type %d is not binary", op)
		}

		ws.r = r
	}

	n := 0
	for {
		if n == len(p) {
			return n, nil
		}

		br, err := ws.r.Read(p[n:])
		n += br
		if err != nil {
			ws.r = nil
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return n, err
		}
	}
}

func (ws *websocketConn) Write(p []byte) (int, error) {
	if err := ws.Conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (ws *websocketConn) Close() error {
	return ws.Conn.Close()
}

func (ws *websocketConn) LocalAddr() net.Addr {
	return ws.Conn.LocalAddr()
}

func (ws *websocketConn) RemoteAddr() net.Addr {
	return ws.Conn.RemoteAddr()
}

func (ws *websocketConn) SetDeadline(t time.Time) error {
	if err := ws.Conn.SetReadDeadline(t); err != nil {
		return err
	}
	return ws.Conn.SetWriteDeadline(t)
}

func (ws *websocketConn) SetReadDeadline(t time.Time) error {
	return ws.Conn.SetReadDeadline(t)
}

func (ws *websocketConn) SetWriteDeadline(t time.Time) error {
	return ws.Conn.SetWriteDeadline(t)
}

func topicMatchesFilter(topic string, filter string) bool {
	if filter == "#" {
		return true
	}

	topicLevels := strings.Split(topic, "/")
	filterLevels := strings.Split(filter, "/")

	for i, f := range filterLevels {
		if f == "#" {
			return true
		}
		if i >= len(topicLevels) {
			return false
		}
		if f != "+" && f != topicLevels[i] {
			return false
		}
	}

	return len(topicLevels) == len(filterLevels)
}
