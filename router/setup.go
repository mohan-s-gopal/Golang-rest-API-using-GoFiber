package router

import (
	"api/util"

	"github.com/gofiber/fiber/v2"
)

type Response struct {
	Error  bool   `json:"error"`
	Msg string `json:"msg"`
}

// USER handles all the user routes
var USER fiber.Router

func Health(c *fiber.Ctx) error {
	return c.SendString("ok!")
}

// SetupRoutes setups all the Routes
func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/health", Health)

	// Login and Registration 
	api.Post("/register", CreateUser) // Sign Up a user
	api.Post("/login", LoginUser)  // Sign In a user
	api.Get("/get-access-token", GetAccessToken)

	// privUser handles all the private user routes that requires authentication
	privUser := api.Group("/user")
	privUser.Use(util.SecureAuth()) // middleware to secure all routes for this group
	privUser.Get("/info", GetUserData)

	// Medicine
	privUser.Get("/medicine", GetMedicine)
	privUser.Post("/medicine", CreateMedicine)
	privUser.Delete("/medicine/:id", DeleteMedicine)
}
