package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	dataFile := getenv("DATA_FILE", "")
	store := LoadStore(dataFile)
	apiToken := getenv("API_TOKEN", "")

	app := fiber.New()

	// Simple bearer auth if API_TOKEN is set
	authMiddleware := func(c *fiber.Ctx) error {
		if apiToken == "" {
			return c.Next()
		}
		if c.Get("Authorization") != "Bearer "+apiToken {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"code":    "unauthorized",
				"message": "Invalid or missing token",
			})
		}
		return c.Next()
	}

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "time": time.Now().UTC()})
	})

	app.Post("/auth/login", func(c *fiber.Ctx) error {
		var creds User
		if err := c.BodyParser(&creds); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		if store.ValidateUser(creds.Username, creds.Password) {
			// Demo token; real impl would issue JWT
			return c.JSON(TokenResponse{AccessToken: apiTokenOrDefault(apiToken), ExpiresAt: time.Now().Add(24 * time.Hour).Unix()})
		}
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"code": "auth_failed", "message": "Invalid credentials"})
	})

	app.Get("/mobile/config", func(c *fiber.Ctx) error {
		apiBase := getenv("API_BASE_URL", "http://localhost:8080")
		uispBase := getenv("UISP_BASE_URL", "http://localhost")
		resp := MobileConfig{
			UispBaseURL:  uispBase,
			APIBaseURL:   apiBase,
			FeatureFlags: map[string]bool{"native_api": true},
			PushRegister: apiBase + "/push/register",
			Environment:  getenv("APP_ENV", "dev"),
			Version:      "0.1.0",
			Banner:       "Demo backend",
		}
		return c.JSON(resp)
	})

	app.Get("/devices", authMiddleware, func(c *fiber.Ctx) error {
		devices := store.ListDevices()
		return c.JSON(DevicesResponse{LastUpdated: time.Now().UnixMilli(), Devices: devices})
	})

	app.Get("/incidents", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.ListIncidents())
	})

	app.Post("/incidents/:id/ack", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var req AckRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		if req.DurationMinutes <= 0 {
			req.DurationMinutes = 30
		}
		inc, ok := store.AckIncident(id, req.DurationMinutes)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
		}
		return c.JSON(inc)
	})

	app.Get("/metrics/devices/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		points := []fiber.Map{
			{"timestamp": time.Now().Add(-5 * time.Minute).Unix(), "latency": 5, "cpu": 20, "ram": 30, "online": true},
			{"timestamp": time.Now().Unix(), "latency": 8, "cpu": 22, "ram": 31, "online": true},
		}
		return c.JSON(fiber.Map{"device_id": id, "points": points})
	})

	app.Post("/push/register", func(c *fiber.Ctx) error {
		var req PushRegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		if req.Token == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "missing_token", "message": "token is required"})
		}
		rid := randomID()
		store.RegisterPush(req)
		log.Printf("Push token registered platform=%s token=%s version=%s locale=%s request_id=%s", req.Platform, req.Token, req.AppVersion, req.Locale, rid)
		return c.JSON(PushRegisterResponse{RequestID: rid, Message: "registered"})
	})

	addr := getenv("API_ADDR", ":8080")
	log.Printf("API listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("failed to start api: %v", err)
	}
}

func apiTokenOrDefault(t string) string {
	if t != "" {
		return t
	}
	return "dev-token"
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func randomID() string {
	return time.Now().Format("20060102T150405.000")
}
