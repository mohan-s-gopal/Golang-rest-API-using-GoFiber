package controllers

import (
	db "api/database"
	"api/models"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// CreateMedicine route adding a medicine into the database
func CreateMedicine(c *fiber.Ctx) error {
	u := new(models.Medicines)
	id := c.Locals("id")
	u.Uuid = fmt.Sprint(id)

	if err := c.BodyParser(u); err != nil {
		print(err.Error())
		return c.JSON(fiber.Map{
			"error": true,
			"input": "Please review your input",
		})
	}

	if err := db.DB.Create(&u).Error; err != nil {
		return c.JSON(fiber.Map{
			"error":   true,
			"general": "Something went wrong, please try again later. 😕",
		})
	}

	resp := Response{Error: true, Msg: "Medicine Added successfully"}

	return c.JSON(resp)
}

func DeleteMedicine(c *fiber.Ctx) error {
	var medicine models.Medicines
	id := c.Params("id")
	
	if err := db.DB.First(&medicine, id).Error; err != nil {
		return c.JSON(fiber.Map{
			"error":   true,
			"general": err.Error(),
		})
	}

	db.DB.Delete(&medicine)

	return c.JSON(Response{
		Error: false, 
		Msg: "Medicine deleted successfully",
	})
}

func GetMedicine(c *fiber.Ctx) error {
	var medicines []models.Medicines
	id := c.Locals("id")

	// result := db.DB.Where("uuid = ?", id).Scan(&medicines)
	result := db.DB.Raw("SELECT * FROM medicines WHERE uuid = ?", id).Scan(&medicines)

	if err := result.Error; err != nil {
		return c.JSON(fiber.Map{
			"error":   true,
			"general": "Something went wrong, please try again later. 😕",
		})
	}
	return c.JSON(medicines)
}
