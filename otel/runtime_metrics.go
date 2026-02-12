package otel

import (
	"context"
	"math"
	runtimemetrics "runtime/metrics"
	"strconv"
	"strings"
	"sync"

	"github.com/bronystylecrazy/ultrastructure/di"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

func UseRuntimeMetrics() di.Node {
	return di.Options(
		di.Provide(NewRuntimeMetrics, Layer("runtime")),
		di.Invoke(func(*RuntimeMetrics) {}),
	)
}

type RuntimeMetrics struct {
	cpu   *runtimeCPUUtilization
	reg   otelmetric.Registration
	mu    sync.Mutex
	state runtimeSampleState
}

type runtimeSampleState struct {
	totalCPU          float64
	idleCPU           float64
	gcCPU             float64
	heapObjectsBytes  uint64
	totalMemoryBytes  uint64
	heapReleasedBytes uint64
	heapFreeBytes     uint64
	heapStacksBytes   uint64
	gcCyclesTotal     uint64
	gcHeapGoalBytes   uint64
	goroutines        uint64
	gcPauseP50        float64
	gcPauseP90        float64
	gcPauseP99        float64
	schedLatencyP50   float64
	schedLatencyP90   float64
	schedLatencyP99   float64
}

func NewRuntimeMetrics(mp *MeterProvider, config Config) (*RuntimeMetrics, error) {
	metrics := &RuntimeMetrics{cpu: &runtimeCPUUtilization{}}
	if !config.Enabled || strings.EqualFold(strings.TrimSpace(config.Metrics.Exporter), "none") {
		return metrics, nil
	}

	meter := mp.Meter("runtime")

	cpuUtilization, err := meter.Float64ObservableGauge(
		"process.cpu.utilization",
		otelmetric.WithDescription("Go process CPU utilization"),
		otelmetric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}
	cpuTotal, err := meter.Float64ObservableGauge(
		"process.runtime.go.cpu.total",
		otelmetric.WithDescription("Total CPU seconds consumed by the Go runtime process"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	cpuGC, err := meter.Float64ObservableGauge(
		"process.runtime.go.cpu.gc",
		otelmetric.WithDescription("CPU seconds consumed by GC"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	heapObjects, err := meter.Int64ObservableGauge(
		"process.runtime.go.memory.heap.objects",
		otelmetric.WithDescription("Heap allocated objects in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}
	memTotal, err := meter.Int64ObservableGauge(
		"process.runtime.go.memory.total",
		otelmetric.WithDescription("Total memory tracked by Go runtime in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}
	heapReleased, err := meter.Int64ObservableGauge(
		"process.runtime.go.memory.heap.released",
		otelmetric.WithDescription("Heap memory released to OS in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}
	heapFree, err := meter.Int64ObservableGauge(
		"process.runtime.go.memory.heap.free",
		otelmetric.WithDescription("Free heap memory in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}
	heapStacks, err := meter.Int64ObservableGauge(
		"process.runtime.go.memory.heap.stacks",
		otelmetric.WithDescription("Heap stack memory in bytes"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	gcCycles, err := meter.Int64ObservableGauge(
		"process.runtime.go.gc.cycles",
		otelmetric.WithDescription("Total completed GC cycles"),
		otelmetric.WithUnit("{gc_cycle}"),
	)
	if err != nil {
		return nil, err
	}
	gcHeapGoal, err := meter.Int64ObservableGauge(
		"process.runtime.go.gc.heap.goal",
		otelmetric.WithDescription("Target heap size for the next GC cycle"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}
	goroutines, err := meter.Int64ObservableGauge(
		"process.runtime.go.goroutines",
		otelmetric.WithDescription("Current goroutine count"),
		otelmetric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return nil, err
	}

	gcPauseP50, err := meter.Float64ObservableGauge(
		"process.runtime.go.gc.pause.p50",
		otelmetric.WithDescription("P50 GC pause duration"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	gcPauseP90, err := meter.Float64ObservableGauge(
		"process.runtime.go.gc.pause.p90",
		otelmetric.WithDescription("P90 GC pause duration"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	gcPauseP99, err := meter.Float64ObservableGauge(
		"process.runtime.go.gc.pause.p99",
		otelmetric.WithDescription("P99 GC pause duration"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	schedP50, err := meter.Float64ObservableGauge(
		"process.runtime.go.scheduler.latency.p50",
		otelmetric.WithDescription("P50 scheduler latency"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	schedP90, err := meter.Float64ObservableGauge(
		"process.runtime.go.scheduler.latency.p90",
		otelmetric.WithDescription("P90 scheduler latency"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	schedP99, err := meter.Float64ObservableGauge(
		"process.runtime.go.scheduler.latency.p99",
		otelmetric.WithDescription("P99 scheduler latency"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	gcPauseBuckets, err := meter.Int64ObservableGauge(
		"process.runtime.go.gc.pause.bucket_count",
		otelmetric.WithDescription("GC pause histogram bucket counts"),
		otelmetric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}
	schedLatencyBuckets, err := meter.Int64ObservableGauge(
		"process.runtime.go.scheduler.latency.bucket_count",
		otelmetric.WithDescription("Scheduler latency histogram bucket counts"),
		otelmetric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}

	samples := []runtimemetrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
		{Name: "/cpu/classes/idle:cpu-seconds"},
		{Name: "/cpu/classes/gc/total:cpu-seconds"},
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/gc/heap/goal:bytes"},
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/gc/pauses:seconds"},
		{Name: "/sched/latencies:seconds"},
	}

	defaultAttrs := DefaultMetricAttributes(config)
	defaultObserveOpts := []otelmetric.ObserveOption(nil)
	if len(defaultAttrs) > 0 {
		defaultObserveOpts = []otelmetric.ObserveOption{
			otelmetric.WithAttributes(defaultAttrs...),
		}
	}
	reg, err := meter.RegisterCallback(func(ctx context.Context, obs otelmetric.Observer) error {
		withAttrs := func(attrs ...attribute.KeyValue) []otelmetric.ObserveOption {
			if len(attrs) == 0 {
				return defaultObserveOpts
			}
			merged := MergeMetricAttributes(defaultAttrs, attrs)
			if len(merged) == 0 {
				return nil
			}
			return []otelmetric.ObserveOption{
				otelmetric.WithAttributes(merged...),
			}
		}

		state := readRuntimeSampleState(samples)

		cpuValue, ok := metrics.cpu.Observe(state.totalCPU, state.idleCPU)
		if ok {
			obs.ObserveFloat64(cpuUtilization, cpuValue, withAttrs()...)
		}
		obs.ObserveFloat64(cpuTotal, state.totalCPU, withAttrs()...)
		obs.ObserveFloat64(cpuGC, state.gcCPU, withAttrs()...)

		obs.ObserveInt64(heapObjects, int64(state.heapObjectsBytes), withAttrs()...)
		obs.ObserveInt64(memTotal, int64(state.totalMemoryBytes), withAttrs()...)
		obs.ObserveInt64(heapReleased, int64(state.heapReleasedBytes), withAttrs()...)
		obs.ObserveInt64(heapFree, int64(state.heapFreeBytes), withAttrs()...)
		obs.ObserveInt64(heapStacks, int64(state.heapStacksBytes), withAttrs()...)

		obs.ObserveInt64(gcCycles, int64(state.gcCyclesTotal), withAttrs()...)
		obs.ObserveInt64(gcHeapGoal, int64(state.gcHeapGoalBytes), withAttrs()...)
		obs.ObserveInt64(goroutines, int64(state.goroutines), withAttrs()...)

		obs.ObserveFloat64(gcPauseP50, state.gcPauseP50, withAttrs()...)
		obs.ObserveFloat64(gcPauseP90, state.gcPauseP90, withAttrs()...)
		obs.ObserveFloat64(gcPauseP99, state.gcPauseP99, withAttrs()...)

		obs.ObserveFloat64(schedP50, state.schedLatencyP50, withAttrs()...)
		obs.ObserveFloat64(schedP90, state.schedLatencyP90, withAttrs()...)
		obs.ObserveFloat64(schedP99, state.schedLatencyP99, withAttrs()...)
		observeHistogramBuckets(obs, gcPauseBuckets, samples[11], defaultAttrs)
		observeHistogramBuckets(obs, schedLatencyBuckets, samples[12], defaultAttrs)

		metrics.mu.Lock()
		metrics.state = state
		metrics.mu.Unlock()
		return nil
	},
		cpuUtilization,
		cpuTotal,
		cpuGC,
		heapObjects,
		memTotal,
		heapReleased,
		heapFree,
		heapStacks,
		gcCycles,
		gcHeapGoal,
		goroutines,
		gcPauseP50,
		gcPauseP90,
		gcPauseP99,
		schedP50,
		schedP90,
		schedP99,
		gcPauseBuckets,
		schedLatencyBuckets,
	)
	if err != nil {
		return nil, err
	}
	metrics.reg = reg

	return metrics, nil
}

func readRuntimeSampleState(samples []runtimemetrics.Sample) runtimeSampleState {
	runtimemetrics.Read(samples)
	state := runtimeSampleState{}

	state.totalCPU = sampleFloat64(samples[0])
	state.idleCPU = sampleFloat64(samples[1])
	state.gcCPU = sampleFloat64(samples[2])
	state.heapObjectsBytes = sampleUint64(samples[3])
	state.totalMemoryBytes = sampleUint64(samples[4])
	state.heapReleasedBytes = sampleUint64(samples[5])
	state.heapFreeBytes = sampleUint64(samples[6])
	state.heapStacksBytes = sampleUint64(samples[7])
	state.gcCyclesTotal = sampleUint64(samples[8])
	state.gcHeapGoalBytes = sampleUint64(samples[9])
	state.goroutines = sampleUint64(samples[10])
	state.gcPauseP50, state.gcPauseP90, state.gcPauseP99 = sampleHistogramQuantiles(samples[11], 0.5, 0.9, 0.99)
	state.schedLatencyP50, state.schedLatencyP90, state.schedLatencyP99 = sampleHistogramQuantiles(samples[12], 0.5, 0.9, 0.99)

	return state
}

func sampleUint64(sample runtimemetrics.Sample) uint64 {
	if sample.Value.Kind() != runtimemetrics.KindUint64 {
		return 0
	}
	return sample.Value.Uint64()
}

func sampleFloat64(sample runtimemetrics.Sample) float64 {
	if sample.Value.Kind() != runtimemetrics.KindFloat64 {
		return 0
	}
	return sample.Value.Float64()
}

func sampleHistogramQuantiles(sample runtimemetrics.Sample, q1, q2, q3 float64) (float64, float64, float64) {
	if sample.Value.Kind() != runtimemetrics.KindFloat64Histogram {
		return 0, 0, 0
	}
	h := sample.Value.Float64Histogram()
	if h == nil {
		return 0, 0, 0
	}
	return histogramQuantile(h, q1), histogramQuantile(h, q2), histogramQuantile(h, q3)
}

func histogramQuantile(h *runtimemetrics.Float64Histogram, q float64) float64 {
	if h == nil || len(h.Counts) == 0 {
		return 0
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}

	var total uint64
	for _, c := range h.Counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	threshold := uint64(math.Ceil(float64(total) * q))
	if threshold == 0 {
		threshold = 1
	}

	var cumulative uint64
	for i, c := range h.Counts {
		cumulative += c
		if cumulative < threshold {
			continue
		}
		if i+1 < len(h.Buckets) {
			upper := h.Buckets[i+1]
			if math.IsInf(upper, 0) {
				if i < len(h.Buckets) {
					prev := h.Buckets[i]
					if !math.IsInf(prev, 0) {
						return prev
					}
				}
				return 0
			}
			return upper
		}
	}

	last := h.Buckets[len(h.Buckets)-1]
	if math.IsInf(last, 0) {
		if len(h.Buckets) > 1 {
			prev := h.Buckets[len(h.Buckets)-2]
			if !math.IsInf(prev, 0) {
				return prev
			}
		}
		return 0
	}
	return last
}

func observeHistogramBuckets(obs otelmetric.Observer, instrument otelmetric.Int64ObservableGauge, sample runtimemetrics.Sample, defaultAttrs []attribute.KeyValue) {
	if sample.Value.Kind() != runtimemetrics.KindFloat64Histogram {
		return
	}
	h := sample.Value.Float64Histogram()
	if h == nil || len(h.Counts) == 0 {
		return
	}
	for i, count := range h.Counts {
		if count == 0 {
			continue
		}
		upper := math.Inf(1)
		if i+1 < len(h.Buckets) {
			upper = h.Buckets[i+1]
		}
		lower := math.Inf(-1)
		if i < len(h.Buckets) {
			lower = h.Buckets[i]
		}
		attrs := MergeMetricAttributes(defaultAttrs, []attribute.KeyValue{
			attribute.String("bucket.lower", formatBucketBound(lower)),
			attribute.String("bucket.upper", formatBucketBound(upper)),
		})
		obs.ObserveInt64(instrument, int64(count), otelmetric.WithAttributes(attrs...))
	}
}

func formatBucketBound(v float64) string {
	switch {
	case math.IsInf(v, -1):
		return "-Inf"
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsNaN(v):
		return "NaN"
	default:
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
}

type runtimeCPUUtilization struct {
	mu          sync.Mutex
	lastBusy    float64
	lastTotal   float64
	initialized bool
}

func (u *runtimeCPUUtilization) Observe(totalCPU, idleCPU float64) (float64, bool) {
	busy := totalCPU - idleCPU

	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.initialized {
		u.lastBusy = busy
		u.lastTotal = totalCPU
		u.initialized = true
		return 0, true
	}
	deltaBusy := busy - u.lastBusy
	deltaTotal := totalCPU - u.lastTotal
	u.lastBusy = busy
	u.lastTotal = totalCPU
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
