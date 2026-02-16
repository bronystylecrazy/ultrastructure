package main

import (
	"github.com/bronystylecrazy/ultrastructure/realtime"
	"github.com/bronystylecrazy/ultrastructure/realtime/mqtt"
	"github.com/gofiber/fiber/v3"
)

type SimpleHandler struct {
}

func NewSimpleHandler() *SimpleHandler {
	return &SimpleHandler{}
}

func (h *SimpleHandler) Handle(router fiber.Router) {
	router.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
}

func (h *SimpleHandler) Subscribe(r mqtt.TopicRegistrar) error {
	return r.Topic("greeting", func(c realtime.Ctx) error {
		return c.Publish("receive", []byte("Hello, World!"), mqtt.NoRetain, mqtt.QoS0)
	})
}
