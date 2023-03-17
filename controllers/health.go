package controllers

import (
	"github.com/gofiber/fiber/v2"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Response struct {
	Error  bool   `json:"error"`
	Msg string `json:"msg"`
}

// USER handles all the user routes
var USER fiber.Router
var tracer = otel.Tracer("API")

func Health(c *fiber.Ctx) error {
	_, span := tracer.Start(c.UserContext(), "HealthCheck")
	defer span.End()
	span.SetAttributes(
		attribute.String("http.method", c.Method()),
		attribute.String("http.url", c.Route().Name),
	)
	return c.SendString("ok!")
}
