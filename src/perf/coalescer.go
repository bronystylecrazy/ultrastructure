// package background

package perf

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"
	"golang.org/x/time/rate"
)

type CoalescerMode int

const (
	TrailingOnly CoalescerMode = iota
	LeadingAndTrailing
	LeadingOnly // immediate send, then suppress for interval; no trailing send
)

type ShutdownMode int

const (
	ShutdownNoop ShutdownMode = iota
	ShutdownSendLatest
)

// Sender is your overridable hook (outer struct implements this).
type Sender[T any] interface {
	Send(ctx context.Context, key string, payload T) error
}

// Coalescer is the interface most callers should depend on.
type Coalescer[T any] interface {
	Update(key string, payload T)
	StopKey(key string)
	Shutdown(ctx context.Context, mode ShutdownMode) // graceful stop (optionally flush)
	Close()                                          // stop background GC (no flush)
}

/****************************************
 * Functional Options                   *
 ****************************************/

type Option func(*config)

type config struct {
	mode            CoalescerMode
	interval        time.Duration
	idleTTL         time.Duration
	globalLimiter   *rate.Limiter
	ctx             context.Context
	minGapAfterSend time.Duration
	lifecycle       fx.Lifecycle
	shutdownMode    ShutdownMode
}

// Defaults so it works out of the box.
func defaultConfig() *config {
	return &config{
		mode:         TrailingOnly,
		interval:     time.Second, // sensible default; override with WithInterval
		idleTTL:      0,           // no idle GC
		ctx:          context.Background(),
		shutdownMode: ShutdownNoop,
	}
}

func WithMode(m CoalescerMode) Option {
	return func(c *config) { c.mode = m }
}

func WithInterval(d time.Duration) Option {
	return func(c *config) { c.interval = d }
}

func WithIdleTTL(d time.Duration) Option {
	return func(c *config) { c.idleTTL = d }
}

func WithGlobalLimiter(l *rate.Limiter) Option {
	return func(c *config) { c.globalLimiter = l }
}

func WithContext(ctx context.Context) Option {
	return func(c *config) { c.ctx = ctx }
}

func WithMinGapAfterSend(d time.Duration) Option {
	return func(c *config) { c.minGapAfterSend = d }
}

// If provided, registers Fx OnStop to call Shutdown(ctx, shutdownMode).
func WithLifecycle(lc fx.Lifecycle) Option {
	return func(c *config) { c.lifecycle = lc }
}

func WithShutdownMode(m ShutdownMode) Option {
	return func(c *config) { c.shutdownMode = m }
}

/****************************************
 * In-memory generic implementation     *
 ****************************************/

type perKey[T any] struct {
	mu         sync.Mutex
	latest     T
	hasData    bool
	timer      *time.Timer
	cooldown   bool
	lastUsed   time.Time
	closed     bool
	gen        uint64    // guard stale timers
	lastSentAt time.Time // for MinGapAfterSend
}

// InMemoryCoalescer is generic over payload T and sender S.
// S must implement Sender[T]. Pass your outer struct as S.
type InMemoryCoalescer[T any, S Sender[T]] struct {
	mode            CoalescerMode
	interval        time.Duration
	idleTTL         time.Duration
	globalLimiter   *rate.Limiter
	minGapAfterSend time.Duration
	ctx             context.Context

	sender S // owner implementing Sender[T]

	mu       sync.Mutex
	keys     map[string]*perKey[T]
	gcStop   chan struct{}
	stopping bool // prevents new timers during Shutdown
}

// NewInMemoryCoalescer is the single constructor using functional options.
// If WithLifecycle is set, it registers an Fx OnStop hook that calls Shutdown(ctx, shutdownMode).
func NewInMemoryCoalescer[T any, S Sender[T]](owner S, opts ...Option) (*InMemoryCoalescer[T, S], error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.interval <= 0 {
		return nil, fmt.Errorf("coalesce: interval must be > 0 (set WithInterval)")
	}

	c := &InMemoryCoalescer[T, S]{
		mode:            cfg.mode,
		interval:        cfg.interval,
		idleTTL:         cfg.idleTTL,
		globalLimiter:   cfg.globalLimiter,
		minGapAfterSend: cfg.minGapAfterSend,
		ctx:             cfg.ctx,
		sender:          owner,
		keys:            make(map[string]*perKey[T]),
	}
	if c.idleTTL > 0 {
		c.gcStop = make(chan struct{})
		go c.gcLoop()
	}

	// Optional Fx lifecycle integration.
	if cfg.lifecycle != nil {
		mode := cfg.shutdownMode
		cfg.lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error { return nil },
			OnStop: func(ctx context.Context) error {
				c.Shutdown(ctx, mode)
				return nil
			},
		})
	}

	return c, nil
}

/*********************
 * Interface methods *
 *********************/

func (c *InMemoryCoalescer[T, S]) Update(key string, payload T) {
	pk := c.getOrCreate(key)

	pk.mu.Lock()
	defer pk.mu.Unlock()

	if pk.closed {
		return
	}

	pk.latest = payload
	pk.hasData = true
	pk.lastUsed = time.Now()

	switch c.mode {
	case TrailingOnly:
		if pk.timer == nil && !c.isStopping() {
			pk.gen++
			myGen := pk.gen
			pk.timer = time.AfterFunc(c.interval, func() {
				c.onTrailingIfCurrent(key, myGen)
			})
		}

	case LeadingAndTrailing:
		if !pk.cooldown {
			// Optional gap: avoid immediate leading right after a trailing send
			if c.minGapAfterSend > 0 && time.Since(pk.lastSentAt) < c.minGapAfterSend {
				pk.cooldown = true
				if pk.timer != nil {
					pk.timer.Stop()
				}
				if !c.isStopping() {
					pk.gen++
					myGen := pk.gen
					pk.timer = time.AfterFunc(c.interval, func() {
						c.onCooldownEndIfCurrent(key, myGen)
					})
				}
				return
			}

			data := pk.latest
			pk.hasData = false
			pk.cooldown = true

			if pk.timer != nil {
				pk.timer.Stop()
			}
			if !c.isStopping() {
				pk.gen++
				myGen := pk.gen
				pk.timer = time.AfterFunc(c.interval, func() {
					c.onCooldownEndIfCurrent(key, myGen)
				})
			}

			go c.sendNow(key, data)
			return
		}
		// already in cooldown → trailing will pick up latest

	case LeadingOnly:
		if !pk.cooldown {
			// Optional gap also applies to leading-only
			if c.minGapAfterSend > 0 && time.Since(pk.lastSentAt) < c.minGapAfterSend {
				pk.cooldown = true
				if pk.timer != nil {
					pk.timer.Stop()
				}
				if !c.isStopping() {
					pk.gen++
					myGen := pk.gen
					pk.timer = time.AfterFunc(c.interval, func() {
						c.clearCooldownIfCurrent(key, myGen)
					})
				}
				return
			}

			data := pk.latest
			pk.hasData = false
			pk.cooldown = true

			if pk.timer != nil {
				pk.timer.Stop()
			}
			if !c.isStopping() {
				pk.gen++
				myGen := pk.gen
				pk.timer = time.AfterFunc(c.interval, func() {
					c.clearCooldownIfCurrent(key, myGen)
				})
			}

			go c.sendNow(key, data)
			return
		}

	default:
		// Fallback to trailing-only
		if pk.timer == nil && !c.isStopping() {
			pk.gen++
			myGen := pk.gen
			pk.timer = time.AfterFunc(c.interval, func() {
				c.onTrailingIfCurrent(key, myGen)
			})
		}
	}
}

func (c *InMemoryCoalescer[T, S]) StopKey(key string) {
	c.mu.Lock()
	pk, ok := c.keys[key]
	if ok {
		delete(c.keys, key)
	}
	c.mu.Unlock()
	if !ok {
		return
	}
	pk.mu.Lock()
	if pk.timer != nil {
		pk.timer.Stop()
		pk.timer = nil
	}
	pk.closed = true
	pk.mu.Unlock()
}

// Shutdown gracefully stops timers & GC and (optionally) sends a final “latest” per key.
func (c *InMemoryCoalescer[T, S]) Shutdown(ctx context.Context, mode ShutdownMode) {
	// Prevent new timers
	c.mu.Lock()
	if c.stopping {
		c.mu.Unlock()
		return
	}
	c.stopping = true
	// Snapshot keys
	keys := make([]string, 0, len(c.keys))
	for k := range c.keys {
		keys = append(keys, k)
	}
	c.mu.Unlock()

	// Stop GC loop
	c.Close()

	// Stop timers, capture payloads if needed, and close keys
	type item[T any] struct {
		key string
		val *T
	}
	var toSend []item[T]

	for _, k := range keys {
		pk := c.get(k)
		if pk == nil {
			continue
		}
		pk.mu.Lock()
		if pk.timer != nil {
			pk.timer.Stop()
			pk.timer = nil
		}

		var chosen *T
		switch mode {
		case ShutdownSendLatest:
			if pk.hasData {
				v := pk.latest
				chosen = &v
				pk.hasData = false
			}
		case ShutdownNoop:
			// nothing
		}

		pk.closed = true
		pk.mu.Unlock()

		if chosen != nil {
			toSend = append(toSend, item[T]{key: k, val: chosen})
		}
	}

	// send outside locks; respect ctx and limiter
	for _, it := range toSend {
		select {
		case <-ctx.Done():
			return
		default:
		}
		c.sendNowWithCtx(ctx, it.key, *it.val)
	}
}

func (c *InMemoryCoalescer[T, S]) Close() {
	if c.gcStop != nil {
		select {
		case <-c.gcStop:
			// already closed
		default:
			close(c.gcStop)
		}
	}
}

/****************
 * Internals    *
 ****************/

func (c *InMemoryCoalescer[T, S]) getOrCreate(key string) *perKey[T] {
	c.mu.Lock()
	defer c.mu.Unlock()
	if pk, ok := c.keys[key]; ok {
		return pk
	}
	pk := &perKey[T]{lastUsed: time.Now()}
	c.keys[key] = pk
	return pk
}

func (c *InMemoryCoalescer[T, S]) get(key string) *perKey[T] {
	c.mu.Lock()
	pk := c.keys[key]
	c.mu.Unlock()
	return pk
}

func (c *InMemoryCoalescer[T, S]) isStopping() bool {
	c.mu.Lock()
	st := c.stopping
	c.mu.Unlock()
	return st
}

// Timer handlers (generation-guarded)

func (c *InMemoryCoalescer[T, S]) onTrailingIfCurrent(key string, want uint64) {
	pk := c.get(key)
	if pk == nil {
		return
	}

	var data *T
	pk.mu.Lock()
	if pk.closed || pk.gen != want {
		pk.mu.Unlock()
		return
	}
	if pk.hasData {
		v := pk.latest
		data = &v
		pk.hasData = false
	}
	pk.timer = nil
	pk.mu.Unlock()

	if data != nil {
		c.sendNow(key, *data)
	}
}

func (c *InMemoryCoalescer[T, S]) onCooldownEndIfCurrent(key string, want uint64) {
	pk := c.get(key)
	if pk == nil {
		return
	}

	var data *T
	pk.mu.Lock()
	if pk.closed || pk.gen != want {
		pk.mu.Unlock()
		return
	}
	if pk.hasData {
		v := pk.latest
		data = &v
		pk.hasData = false
	}
	pk.cooldown = false
	pk.timer = nil
	pk.mu.Unlock()

	if data != nil {
		c.sendNow(key, *data)
	}
}

func (c *InMemoryCoalescer[T, S]) clearCooldownIfCurrent(key string, want uint64) {
	pk := c.get(key)
	if pk == nil {
		return
	}
	pk.mu.Lock()
	if pk.closed || pk.gen != want {
		pk.mu.Unlock()
		return
	}
	pk.cooldown = false
	pk.timer = nil
	pk.mu.Unlock()
}

// Sending

func (c *InMemoryCoalescer[T, S]) sendNow(key string, payload T) {
	// Best-effort using constructor context.
	if c.globalLimiter != nil {
		_ = c.globalLimiter.Wait(c.ctx)
	}
	_ = c.sender.Send(c.ctx, key, payload)

	// Track per-key last send time for MinGapAfterSend
	if c.minGapAfterSend > 0 {
		if pk := c.get(key); pk != nil {
			pk.mu.Lock()
			pk.lastSentAt = time.Now()
			pk.mu.Unlock()
		}
	}
}

// Variant that honors the passed context (used during Shutdown)
func (c *InMemoryCoalescer[T, S]) sendNowWithCtx(ctx context.Context, key string, payload T) {
	if c.globalLimiter != nil {
		// Pace with constructor context limiter, but honor stop ctx deadline for the send itself.
		type token struct{}
		got := make(chan token, 1)
		go func() {
			_ = c.globalLimiter.Wait(c.ctx)
			got <- token{}
		}()
		select {
		case <-ctx.Done():
			return
		case <-got:
		}
	}
	_ = c.sender.Send(ctx, key, payload)

	if c.minGapAfterSend > 0 {
		if pk := c.get(key); pk != nil {
			pk.mu.Lock()
			pk.lastSentAt = time.Now()
			pk.mu.Unlock()
		}
	}
}

// GC

func (c *InMemoryCoalescer[T, S]) gcLoop() {
	period := c.idleTTL / 2
	if period < 10*time.Second {
		period = 10 * time.Second
	}
	t := time.NewTicker(period)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.gcOnce()
		case <-c.gcStop:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *InMemoryCoalescer[T, S]) gcOnce() {
	if c.idleTTL <= 0 {
		return
	}
	now := time.Now()
	var victims []string

	c.mu.Lock()
	for key, pk := range c.keys {
		pk.mu.Lock()
		idle := now.Sub(pk.lastUsed) >= c.idleTTL
		draining := pk.cooldown || pk.timer != nil || pk.hasData
		closed := pk.closed
		pk.mu.Unlock()

		if !closed && idle && !draining {
			victims = append(victims, key)
		}
	}
	for _, key := range victims {
		if pk, ok := c.keys[key]; ok {
			delete(c.keys, key)
			pk.mu.Lock()
			if pk.timer != nil {
				pk.timer.Stop()
				pk.timer = nil
			}
			pk.closed = true
			pk.mu.Unlock()
		}
	}
	c.mu.Unlock()
}
