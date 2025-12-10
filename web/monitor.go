package web

import (
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

type MonitorHandler interface {
	Handler
}

var NopMonitorHandler MonitorHandler = &NopHandler{}

type monitorHandler struct{}

func NewMonitorHandler() MonitorHandler {
	return &monitorHandler{}
}

func (h *monitorHandler) Handle(app App) {
	app.Get("/metrics", monitor.New())
}
