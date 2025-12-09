package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// In-memory stores (lightweight placeholder until real persistence is added)
type stores struct {
	mu        sync.Mutex
	devices   []Device
	incidents []Incident
	pushTokens []PushRegisterRequest
}

type MobileConfig struct {
	UispBaseURL  string          `json:"uisp_base_url"`
	APIBaseURL   string          `json:"api_base_url"`
	FeatureFlags map[string]bool `json:"feature_flags"`
	PushRegister string          `json:"push_register_url"`
	Environment  string          `json:"environment"`
	Version      string          `json:"version"`
	Banner       string          `json:"banner"`
}

type Device struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	SiteID    string   `json:"site_id"`
	Online    bool     `json:"online"`
	LatencyMs *float64 `json:"latency_ms"`
	AckUntil  *int64   `json:"ack_until"`
}

type Incident struct {
	ID        string  `json:"id"`
	DeviceID  string  `json:"device_id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Started   string  `json:"started_at"`
	Resolved  *string `json:"resolved_at"`
	AckUntil  *string `json:"ack_until"`
}

type PushRegisterRequest struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	Locale     string `json:"locale"`
}

type PushRegisterResponse struct {
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
}

type DevicesResponse struct {
	LastUpdated int64    `json:"last_updated"`
	Devices     []Device `json:"devices"`
}

type AckRequest struct {
	DurationMinutes int `json:"duration_minutes"`
}

func main() {
	app := fiber.New()
	store := seedStore()

	apiToken := getenv("API_TOKEN", "")
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
		store.mu.Lock()
		defer store.mu.Unlock()
		return c.JSON(DevicesResponse{LastUpdated: time.Now().UnixMilli(), Devices: store.devices})
	})

	app.Get("/incidents", authMiddleware, func(c *fiber.Ctx) error {
		store.mu.Lock()
		defer store.mu.Unlock()
		return c.JSON(store.incidents)
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
		store.mu.Lock()
		defer store.mu.Unlock()
		for i := range store.incidents {
			if store.incidents[i].ID == id {
				until := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute).UTC().Format(time.RFC3339)
				store.incidents[i].AckUntil = &until
				return c.JSON(store.incidents[i])
			}
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
	})

	app.Get("/metrics/devices/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		// Placeholder metrics
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
		store.mu.Lock()
		store.pushTokens = append(store.pushTokens, req)
		store.mu.Unlock()
		log.Printf("Push token registered platform=%s token=%s version=%s locale=%s request_id=%s", req.Platform, req.Token, req.AppVersion, req.Locale, rid)
		return c.JSON(PushRegisterResponse{RequestID: rid, Message: "registered"})
	})

	addr := getenv("API_ADDR", ":8080")
	log.Printf("API listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("failed to start api: %v", err)
	}
}

func seedStore() *stores {
	lat12 := 12.0
	lat3 := 3.0
	devs := []Device{
		{ID: "gw-1", Name: "Gateway-1", Role: "gateway", SiteID: "site-1", Online: true, LatencyMs: &lat12},
		{ID: "ap-1", Name: "AP-1", Role: "ap", SiteID: "site-1", Online: false, LatencyMs: nil},
		{ID: "sw-1", Name: "Switch-1", Role: "switch", SiteID: "site-1", Online: true, LatencyMs: &lat3},
	}
	now := time.Now().UTC().Format(time.RFC3339)
	inc := []Incident{
		{ID: "inc-1", DeviceID: "ap-1", Type: "offline", Severity: "critical", Started: now, Resolved: nil, AckUntil: nil},
	}
	return &stores{devices: devs, incidents: inc, pushTokens: []PushRegisterRequest{}}
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
