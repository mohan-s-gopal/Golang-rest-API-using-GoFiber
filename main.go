package main

import (
	"log"

	"api/database"
	"api/router"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CreateServer creates a new Fiber instance
func CreateServer() *fiber.App {
	app := fiber.New()

	return app
}

func main() {
	// Connect to Postgres 
	database.ConnectSqlite3()
	app := CreateServer()

	app.Use(cors.New())

	router.SetupRoutes(app)

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	log.Fatal(app.Listen(":3001"))
	log.Print("Application started ...")
}
