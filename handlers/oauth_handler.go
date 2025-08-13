package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GoogleUserInfo untuk menampung data dari Google API
type GoogleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GoogleLogin me-redirect pengguna ke halaman login Google
func GoogleLogin(c *fiber.Ctx) error {
	if config.GoogleOAuthConfig == nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Google OAuth not configured on server"})
	}
	url := config.GoogleOAuthConfig.AuthCodeURL("randomstate")
	return c.Redirect(url, http.StatusTemporaryRedirect)
}

// GoogleCallback menangani response dari Google setelah login
func GoogleCallback(c *fiber.Ctx) error {
	// ... (kode validasi state dan pengambilan token tetap sama) ...
	if c.Query("state") != "randomstate" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid state"})
	}
	code := c.Query("code")
	token, err := config.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth Exchange failed: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to exchange token with Google"})
	}
	res, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get user info from Google"})
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var userInfo GoogleUserInfo
	json.Unmarshal(body, &userInfo)
	// ... (akhir dari kode yang tidak berubah) ...

	collection := config.GetCollection("users")
	var user models.User
	err = collection.FindOne(context.Background(), bson.M{"email": userInfo.Email}).Decode(&user)

	if err == mongo.ErrNoDocuments {
		// --- PERBAIKAN: Cari RoleID default untuk user baru ---
		roleCollection := config.GetCollection("roles")
		var defaultRole models.Role
		// Kita cari role 'Viewer' atau 'Editor' atau apa pun yang Anda tetapkan sebagai default
		err := roleCollection.FindOne(context.Background(), bson.M{"name": "Viewer"}).Decode(&defaultRole)
		if err != nil {
			// Jika role default tidak ada, ini adalah error konfigurasi server
			log.Printf("FATAL: Default role 'Viewer' not found in database.")
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Server is not configured correctly, default role missing."})
		}
		// ----------------------------------------------------

		now := time.Now()
		newUser := models.User{
			BaseModel: models.BaseModel{
				ID:         primitive.NewObjectID(),
				CreatedOn:  now,
				ModifiedOn: now,
			},
			Username: userInfo.Name,
			Email:    userInfo.Email,
			// --- PERBAIKAN: Gunakan RoleID dari role default ---
			RoleID: defaultRole.ID,
			// ------------------------------------------------
			IsActive:   true,
			Provider:   "google",
			ProviderID: userInfo.ID,
		}
		_, err = collection.InsertOne(context.Background(), newUser)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create new user"})
		}
		user = newUser
	} else if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Database lookup error"})
	} else {
		if user.Provider != "google" {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "This email is already registered using username/password"})
		}
	}

	return generateLoginResponse(c, user)
}
