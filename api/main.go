package main

import (
    "log"
    "net/http"
    "os"
    "time"

    "github.com/gofiber/fiber/v2"
)

type MobileConfig struct {
    UispBaseURL  string            `json:"uisp_base_url"`
    APIBaseURL   string            `json:"api_base_url"`
    FeatureFlags map[string]bool   `json:"feature_flags"`
    PushRegister string            `json:"push_register_url"`
    Environment  string            `json:"environment"`
    Version      string            `json:"version"`
    Banner       string            `json:"banner"`
}

type Device struct {
    ID        string  `json:"id"`
    Name      string  `json:"name"`
    Role      string  `json:"role"`
    SiteID    string  `json:"site_id"`
    Online    bool    `json:"online"`
    LatencyMs *float64 `json:"latency_ms"`
    AckUntil  *int64  `json:"ack_until"`
}

type Incident struct {
    ID       string `json:"id"`
    DeviceID string `json:"device_id"`
    Type     string `json:"type"`
    Severity string `json:"severity"`
    Started  string `json:"started_at"`
    Resolved *string `json:"resolved_at"`
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

func main() {
    app := fiber.New()

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

    app.Get("/devices", func(c *fiber.Ctx) error {
        devices := []Device{
            {ID: "gw-1", Name: "Gateway-1", Role: "gateway", SiteID: "site-1", Online: true, LatencyMs: ptrFloat(12)},
            {ID: "ap-1", Name: "AP-1", Role: "ap", SiteID: "site-1", Online: false, LatencyMs: ptrFloat(0)},
            {ID: "sw-1", Name: "Switch-1", Role: "switch", SiteID: "site-1", Online: true, LatencyMs: ptrFloat(3)},
        }
        return c.JSON(DevicesResponse{LastUpdated: time.Now().UnixMilli(), Devices: devices})
    })

    app.Get("/incidents", func(c *fiber.Ctx) error {
        now := time.Now().UTC().Format(time.RFC3339)
        incidents := []Incident{
            {ID: "inc-1", DeviceID: "ap-1", Type: "offline", Severity: "critical", Started: now, Resolved: nil},
        }
        return c.JSON(incidents)
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
        log.Printf("Push token registered platform=%s token=%s version=%s locale=%s request_id=%s", req.Platform, req.Token, req.AppVersion, req.Locale, rid)
        return c.JSON(PushRegisterResponse{RequestID: rid, Message: "registered"})
    })

    addr := getenv("API_ADDR", ":8080")
    log.Printf("API listening on %s", addr)
    if err := app.Listen(addr); err != nil {
        log.Fatalf("failed to start api: %v", err)
    }
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

func ptrFloat(v float64) *float64 { return &v }
