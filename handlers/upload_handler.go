package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UploadImage menangani upload file gambar
func UploadImage(c *fiber.Ctx) error {
	// 1. Ambil file dari form request
	file, err := c.FormFile("image") // "image" harus cocok dengan key di FormData frontend
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Gagal menerima file: " + err.Error()})
	}

	// 2. Buat nama file unik untuk mencegah tumpang tindih
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), uuid.New().String(), ext)

	// 3. Tentukan path tujuan untuk menyimpan file
	// Pastikan folder './public/uploads' sudah ada
	uploadDir := "./public/uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755) // Buat direktori jika belum ada
	}
	savePath := filepath.Join(uploadDir, newFileName)

	// 4. Simpan file ke direktori di server
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file: " + err.Error()})
	}

	// 5. Buat URL yang bisa diakses publik
	// Ambil BASE_URL dari environment variable, atau fallback ke localhost untuk development
	baseUrl := os.Getenv("BASE_URL")
	if baseUrl == "" {
		baseUrl = fmt.Sprintf("http://%s", c.Hostname()) // c.Hostname() akan resolve ke 'localhost:3033'
	}
	fileUrl := fmt.Sprintf("%s/uploads/%s", baseUrl, newFileName)

	// 6. Kirim kembali response JSON berisi URL gambar
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url": fileUrl,
	})
}
