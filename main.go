package mdp_project_backend

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"message": "Hello MDP Project Backend!",
		})
	})

	app.Listen(":3033")
}
