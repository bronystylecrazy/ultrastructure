package otel

import (
	"context"
	"time"

	usotel "go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type MeterProvider struct {
	*sdkmetric.MeterProvider
}

func NewMeterProvider(resource *resource.Resource, exporter sdkmetric.Exporter, config Config) (*MeterProvider, error) {
	if exporter == nil {
		mp := &MeterProvider{sdkmetric.NewMeterProvider()}
		usotel.SetMeterProvider(mp.MeterProvider)
		return mp, nil
	}
	interval := time.Duration(config.Metrics.Tuning.ExportIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = 10 * time.Second
	}
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(interval))
	mp := &MeterProvider{sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(reader),
	)}
	usotel.SetMeterProvider(mp.MeterProvider)
	return mp, nil
}

func (mp *MeterProvider) Stop(ctx context.Context) error {
	return mp.Shutdown(ctx)
}
