package main

import (
	"context"
	"log"
	"sync"

	"api/controllers"
	"api/database"
	"api/util"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var resource *sdkresource.Resource
var initResourcesOnce sync.Once

func initResource() *sdkresource.Resource {
	initResourcesOnce.Do(func() {
		extraResources, _ := sdkresource.New(
			context.Background(),
			sdkresource.WithOS(),
			sdkresource.WithProcess(),
			sdkresource.WithContainer(),
			sdkresource.WithHost(),
			sdkresource.WithProcessExecutableName(),
			sdkresource. WithAttributes(
				semconv.ServiceNameKey.String("m-app"),
			),			
		)
		resource, _ = sdkresource.Merge(
			sdkresource.Default(),
			extraResources,
		)
	})
	return resource
}

func initTracerProvider() *sdktrace.TracerProvider {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		log.Fatalf("new otlp trace exporter failed: %v", err)
	}
	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	// bsp := sdktrace.NewBatchSpanProcessor(exporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		// sdktrace.WithSpanProcessor(bsp),
		// sdktrace.WithResource(
		// 	sdkresource.NewWithAttributes(
		// 		semconv.SchemaURL,
		// 		semconv.ServiceNameKey.String("my-service"),
		// 	)),
		sdktrace.WithResource(initResource()),
	)
	

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func main() {
	tp :=initTracerProvider()

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	app := fiber.New()

	// Connect to Postgres 
	database.ConnectSqlite3()

	// Default middleware config
	app.Use(otelfiber.Middleware())
	app.Use(logger.New())
	app.Use(cors.New())

	// routes
	routes(app)

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	log.Print("Application started ...")
	app.Listen("localhost:3001")
}

func routes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", controllers.Health)

	// Login and Registration 
	api.Post("/register", controllers.CreateUser) // Sign Up a user
	api.Post("/login", controllers.LoginUser)  // Sign In a user
	api.Get("/get-access-token", controllers.GetAccessToken)

	// privUser handles all the private user routes that requires authentication
	privUser := api.Group("/user")
	privUser.Use(util.SecureAuth()) // middleware to secure all routes for this group
	privUser.Get("/info", controllers.GetUserData)

	// Medicine
	privUser.Get("/medicine", controllers.GetMedicine)
	privUser.Post("/medicine", controllers.CreateMedicine)
	privUser.Delete("/medicine/:id", controllers.DeleteMedicine)
}