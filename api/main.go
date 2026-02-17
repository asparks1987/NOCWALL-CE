package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dataFile := getenv("DATA_FILE", "")
	store := LoadStore(dataFile)
	apiToken := getenv("API_TOKEN", "")

	app := fiber.New()

	// Simple bearer auth if API_TOKEN is set.
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
			return c.JSON(TokenResponse{AccessToken: apiTokenOrDefault(apiToken), ExpiresAt: time.Now().Add(24 * time.Hour).Unix()})
		}
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"code": "auth_failed", "message": "Invalid credentials"})
	})

	app.Get("/mobile/config", func(c *fiber.Ctx) error {
		apiBase := getenv("API_BASE_URL", "http://localhost:8080")
		uispBase := getenv("UISP_BASE_URL", "http://localhost")
		resp := MobileConfig{
			UispBaseURL: uispBase,
			APIBaseURL:  apiBase,
			FeatureFlags: map[string]bool{
				"native_api":      true,
				"agent_ingest":    true,
				"events_ingest":   true,
				"cloud_multi_tenant_stub": true,
			},
			PushRegister: apiBase + "/push/register",
			Environment:  getenv("APP_ENV", "dev"),
			Version:      "0.2.0",
			Banner:       "NOCWALL-CE API (stubbed SaaS ingestion)",
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

	app.Get("/agents", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"agents": store.ListAgents(), "stub": true})
	})

	app.Post("/agents/register", authMiddleware, func(c *fiber.Ctx) error {
		var req AgentRegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		agent := store.RegisterAgent(req)
		logger.Info("agent_registered", "agent_id", agent.ID, "site_id", agent.SiteID, "version", agent.Version)
		return c.JSON(fiber.Map{"agent": agent, "stub": true})
	})

	app.Post("/telemetry/ingest", authMiddleware, func(c *fiber.Ctx) error {
		var req TelemetryIngestRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		device, incident, ok := store.IngestTelemetry(req)
		if !ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "missing_device_id", "message": "device_id is required"})
		}
		logger.Info("telemetry_ingested", "device_id", device.ID, "event_type", req.EventType, "source", req.Source, "online", device.Online)
		return c.JSON(TelemetryIngestResponse{Accepted: true, Device: device, Incident: incident, Stub: true})
	})

	app.Post("/events/ingest", authMiddleware, func(c *fiber.Ctx) error {
		var req EventIngestRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		eventType := strings.ToLower(strings.TrimSpace(req.Type))
		if eventType == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "missing_type", "message": "type is required"})
		}

		online := true
		if eventType == "device_down" || eventType == "offline" {
			online = false
		}
		telemetry := TelemetryIngestRequest{
			Source:    "events_endpoint",
			EventType: eventType,
			DeviceID:  req.DeviceID,
			Device:    req.Device,
			SiteID:    req.Site,
			Online:    &online,
			Message:   req.Message,
		}
		device, incident, ok := store.IngestTelemetry(telemetry)
		if !ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "missing_device_id", "message": "device_id is required"})
		}
		logger.Info("event_ingested", "type", eventType, "device_id", device.ID, "online", device.Online)
		return c.JSON(TelemetryIngestResponse{Accepted: true, Device: device, Incident: incident, Stub: true})
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
		logger.Info("push_registered", "platform", req.Platform, "app_version", req.AppVersion, "locale", req.Locale, "request_id", rid)
		return c.JSON(PushRegisterResponse{RequestID: rid, Message: "registered"})
	})

	addr := getenv("API_ADDR", ":8080")
	logger.Info("api_listening", "addr", addr, "data_file", dataFile)
	if err := app.Listen(addr); err != nil {
		logger.Error("api_start_failed", "error", err)
		os.Exit(1)
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
