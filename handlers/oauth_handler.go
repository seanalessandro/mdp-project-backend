package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mdp-project-backend/config"
	"mdp-project-backend/models"
	"mdp-project-backend/utils"
	"net/http"
	"net/url" // Pastikan package ini di-import
	"os"      // Pastikan package ini di-import
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

// GoogleCallback menangani response dari Google dan me-redirect ke Frontend
func GoogleCallback(c *fiber.Ctx) error {
	// 1. Validasi state dan tukar token dengan Google
	if c.Query("state") != "randomstate" {
		return c.Status(http.StatusBadRequest).SendString("Invalid state")
	}
	code := c.Query("code")
	token, err := config.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth Exchange failed: %v", err)
		return c.Status(http.StatusInternalServerError).SendString("Failed to exchange token with Google")
	}

	// 2. Ambil info user dari Google
	res, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Failed to get user info from Google")
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var userInfo GoogleUserInfo
	json.Unmarshal(body, &userInfo)

	// 3. Cari atau buat user di database
	collection := config.GetCollection("users")
	var user models.User
	err = collection.FindOne(context.Background(), bson.M{"email": userInfo.Email}).Decode(&user)

	if err == mongo.ErrNoDocuments {
		// Cari RoleID default untuk user baru
		roleCollection := config.GetCollection("roles")
		var defaultRole models.Role
		// Pastikan role 'viewer' (huruf kecil) ada di database Anda
		err := roleCollection.FindOne(context.Background(), bson.M{"name": "viewer"}).Decode(&defaultRole)
		if err != nil {
			log.Printf("FATAL: Default role 'viewer' not found in database.")
			return c.Status(http.StatusInternalServerError).SendString("Server is not configured correctly, default role missing.")
		}

		now := time.Now()
		newUser := models.User{
			BaseModel: models.BaseModel{
				ID:         primitive.NewObjectID(),
				CreatedOn:  now,
				ModifiedOn: now,
			},
			Username:   userInfo.Name,
			Email:      userInfo.Email,
			RoleID:     defaultRole.ID,
			IsActive:   true,
			Provider:   "google",
			ProviderID: userInfo.ID,
		}
		_, err = collection.InsertOne(context.Background(), newUser)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString("Failed to create new user")
		}
		user = newUser
	} else if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Database lookup error")
	} else if user.Provider != "google" {
		return c.Status(http.StatusConflict).SendString("This email is already registered using username/password")
	}

	// --- PERUBAHAN UTAMA: Lakukan Redirect ke Frontend ---

	// 4. Ambil detail role dari user yang login
	roleCollection := config.GetCollection("roles")
	var role models.Role
	err = roleCollection.FindOne(context.Background(), bson.M{"_id": user.RoleID}).Decode(&role)
	if err != nil {
		log.Printf("FATAL: Could not find role for user %s", user.Username)
		return c.Status(500).SendString("Server configuration error: user role not found")
	}

	// 5. Generate token JWT internal
	jwtToken, err := utils.GenerateJWT(user.Username, role.Name, user.ID.Hex())
	if err != nil {
		return c.Status(500).SendString("Failed to generate session token")
	}

	// 6. Siapkan data untuk dikirim ke frontend
	userJSON, _ := json.Marshal(user)
	roleJSON, _ := json.Marshal(role)

	// 7. Bangun URL redirect
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Fallback jika tidak ada di .env
	}

	redirectURL := frontendURL + "/auth/callback?" +
		"token=" + url.QueryEscape(jwtToken) +
		"&user=" + url.QueryEscape(string(userJSON)) +
		"&role=" + url.QueryEscape(string(roleJSON))

	// 8. Lakukan redirect ke frontend
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}
