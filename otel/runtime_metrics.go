package otel

import (
	"context"
	"runtime"
	runtimemetrics "runtime/metrics"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/di"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func UseRuntimeMetrics() di.Node {
	return di.Options(
		di.Provide(NewRuntimeMetrics, Layer("runtime")),
		di.Invoke(func(*RuntimeMetrics) {}),
	)
}

type RuntimeMetrics struct {
	cpu *runtimeCPUUtilization
}

func NewRuntimeMetrics(mp *MeterProvider, config Config) (*RuntimeMetrics, error) {
	metrics := &RuntimeMetrics{
		cpu: &runtimeCPUUtilization{},
	}
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Metrics.Exporter), "none") {
		return metrics, nil
	}

	meter := mp.Meter("runtime")

	_, err := meter.Float64ObservableGauge(
		"process.cpu.utilization",
		otelmetric.WithDescription("Go process CPU utilization"),
		otelmetric.WithUnit("1"),
		otelmetric.WithFloat64Callback(func(ctx context.Context, o otelmetric.Float64Observer) error {
			value, ok := metrics.cpu.Observe()
			if ok {
				o.Observe(value)
			}
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"memory.usage",
		otelmetric.WithDescription("Current process memory allocation in bytes"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(observeMemStat(func(mem *runtime.MemStats) int64 { return int64(mem.Alloc) })),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"memory.heap.usage",
		otelmetric.WithDescription("Current heap allocation in bytes"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(observeMemStat(func(mem *runtime.MemStats) int64 { return int64(mem.HeapAlloc) })),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"memory.sys",
		otelmetric.WithDescription("Memory obtained from the OS in bytes"),
		otelmetric.WithUnit("By"),
		otelmetric.WithInt64Callback(observeMemStat(func(mem *runtime.MemStats) int64 { return int64(mem.Sys) })),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"process.runtime.go.goroutines",
		otelmetric.WithDescription("Current number of goroutines"),
		otelmetric.WithUnit("{goroutine}"),
		otelmetric.WithInt64Callback(func(ctx context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"process.runtime.go.gomaxprocs",
		otelmetric.WithDescription("Current GOMAXPROCS setting"),
		otelmetric.WithUnit("{thread}"),
		otelmetric.WithInt64Callback(func(ctx context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(runtime.GOMAXPROCS(0)))
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"process.runtime.go.gc.cycles",
		otelmetric.WithDescription("Total completed GC cycles"),
		otelmetric.WithUnit("{gc}"),
		otelmetric.WithInt64Callback(observeMemStat(func(mem *runtime.MemStats) int64 { return int64(mem.NumGC) })),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"process.runtime.go.gc.pause_total_ns",
		otelmetric.WithDescription("Total GC pause time in nanoseconds"),
		otelmetric.WithUnit("ns"),
		otelmetric.WithInt64Callback(observeMemStat(func(mem *runtime.MemStats) int64 { return int64(mem.PauseTotalNs) })),
	)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func observeMemStat(selector func(mem *runtime.MemStats) int64) func(context.Context, otelmetric.Int64Observer) error {
	return func(ctx context.Context, o otelmetric.Int64Observer) error {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		o.Observe(selector(&mem))
		return nil
	}
}

type runtimeCPUUtilization struct {
	mu          sync.Mutex
	lastBusy    float64
	lastTotal   float64
	initialized bool
}

func (u *runtimeCPUUtilization) Observe() (float64, bool) {
	samples := []runtimemetrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
		{Name: "/cpu/classes/idle:cpu-seconds"},
	}
	runtimemetrics.Read(samples)
	if len(samples) != 2 {
		return 0, false
	}
	if samples[0].Value.Kind() != runtimemetrics.KindFloat64 || samples[1].Value.Kind() != runtimemetrics.KindFloat64 {
		return 0, false
	}
	total := samples[0].Value.Float64()
	idle := samples[1].Value.Float64()
	busy := total - idle

	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.initialized {
		u.lastBusy = busy
		u.lastTotal = total
		u.initialized = true
		return 0, true
	}
	deltaBusy := busy - u.lastBusy
	deltaTotal := total - u.lastTotal
	u.lastBusy = busy
	u.lastTotal = total
	if deltaTotal <= 0 {
		return 0, true
	}
	utilization := deltaBusy / deltaTotal
	if utilization < 0 {
		utilization = 0
	}
	if utilization > 1 {
		utilization = 1
	}
	return utilization, true
}
