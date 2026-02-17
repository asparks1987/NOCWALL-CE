package main

import "time"

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
	Source    string   `json:"source,omitempty"`
	LastSeen  int64    `json:"last_seen,omitempty"`
}

type Incident struct {
	ID       string  `json:"id"`
	DeviceID string  `json:"device_id"`
	Type     string  `json:"type"`
	Severity string  `json:"severity"`
	Started  string  `json:"started_at"`
	Resolved *string `json:"resolved_at"`
	AckUntil *string `json:"ack_until"`
	Message  string  `json:"message,omitempty"`
	Source   string  `json:"source,omitempty"`
}

type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SiteID       string   `json:"site_id,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastSeen     int64    `json:"last_seen"`
	Status       string   `json:"status"`
}

type AgentRegisterRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SiteID       string   `json:"site_id,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type TelemetryIngestRequest struct {
	Source    string   `json:"source,omitempty"`
	AgentID   string   `json:"agent_id,omitempty"`
	EventType string   `json:"event_type,omitempty"`
	DeviceID  string   `json:"device_id"`
	Device    string   `json:"device,omitempty"`
	Role      string   `json:"role,omitempty"`
	SiteID    string   `json:"site_id,omitempty"`
	Online    *bool    `json:"online,omitempty"`
	LatencyMs *float64 `json:"latency_ms,omitempty"`
	Message   string   `json:"message,omitempty"`
}

type EventIngestRequest struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	Device   string `json:"device,omitempty"`
	Site     string `json:"site,omitempty"`
	Message  string `json:"message,omitempty"`
}

type TelemetryIngestResponse struct {
	Accepted bool      `json:"accepted"`
	Device   Device    `json:"device"`
	Incident *Incident `json:"incident,omitempty"`
	Stub     bool      `json:"stub"`
}

type SourcePollRequest struct {
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Demo    bool   `json:"demo,omitempty"`
	Retries int    `json:"retries,omitempty"`
}

type SourcePollResponse struct {
	Source           string `json:"source"`
	Cursor           string `json:"cursor"`
	Fetched          int    `json:"fetched"`
	Normalized       int    `json:"normalized"`
	Emitted          int    `json:"emitted"`
	Deduped          int    `json:"deduped"`
	Ingested         int    `json:"ingested"`
	IncidentsCreated int    `json:"incidents_created"`
	Backfill         bool   `json:"backfill"`
	Demo             bool   `json:"demo"`
	DurationMs       int64  `json:"duration_ms"`
	Stub             bool   `json:"stub"`
	Error            string `json:"error,omitempty"`
}

type SourceStatus struct {
	Source         string `json:"source"`
	LastPollAt     string `json:"last_poll_at,omitempty"`
	LastCursor     string `json:"last_cursor,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	LastFetched    int    `json:"last_fetched"`
	LastNormalized int    `json:"last_normalized"`
	LastEmitted    int    `json:"last_emitted"`
	Demo           bool   `json:"demo"`
	Stub           bool   `json:"stub"`
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

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func seedDevices() []Device {
	lat12 := 12.0
	lat3 := 3.0
	now := time.Now().UnixMilli()
	return []Device{
		{ID: "gw-1", Name: "Gateway-1", Role: "gateway", SiteID: "site-1", Online: true, LatencyMs: &lat12, Source: "seed", LastSeen: now},
		{ID: "ap-1", Name: "AP-1", Role: "ap", SiteID: "site-1", Online: false, LatencyMs: nil, Source: "seed", LastSeen: now},
		{ID: "sw-1", Name: "Switch-1", Role: "switch", SiteID: "site-1", Online: true, LatencyMs: &lat3, Source: "seed", LastSeen: now},
	}
}

func seedIncidents() []Incident {
	now := time.Now().UTC().Format(time.RFC3339)
	return []Incident{
		{ID: "inc-1", DeviceID: "ap-1", Type: "offline", Severity: "critical", Started: now, Source: "seed"},
	}
}
