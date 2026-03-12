package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VendorConnector provides a generic HTTP-backed source connector used for
// Cisco/Juniper beta adapters where endpoint/auth details are deployment-configured.
type VendorConnector struct {
	source      string
	vendorLabel string
	baseURL     string
	token       string
	devicesPath string
	authScheme  string
	client      *http.Client

	mu        sync.RWMutex
	status    SourceStatus
	lastKnown map[string]bool
	seen      map[string]int64
}

func NewVendorConnector(source, vendorLabel, baseURL, token, devicesPath, authScheme string) *VendorConnector {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		source = "vendor"
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if devicesPath == "" {
		devicesPath = "/api/v1/devices"
	}
	if !strings.HasPrefix(devicesPath, "/") {
		devicesPath = "/" + devicesPath
	}
	authScheme = strings.TrimSpace(strings.ToLower(authScheme))
	if authScheme == "" {
		authScheme = "bearer"
	}
	vendorLabel = strings.TrimSpace(vendorLabel)
	if vendorLabel == "" {
		vendorLabel = connectorLabelFromSource(source)
	}

	return &VendorConnector{
		source:      source,
		vendorLabel: vendorLabel,
		baseURL:     baseURL,
		token:       strings.TrimSpace(token),
		devicesPath: devicesPath,
		authScheme:  authScheme,
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
		status:    SourceStatus{Source: source, Stub: true},
		lastKnown: map[string]bool{},
		seen:      map[string]int64{},
	}
}

func (v *VendorConnector) Name() string {
	return v.source
}

func (v *VendorConnector) Status() SourceStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.status
}

func (v *VendorConnector) Poll(ctx context.Context, req SourcePollRequest) (sourcePollBatch, error) {
	start := time.Now()
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 200
	}
	if req.Retries < 0 {
		req.Retries = 0
	}
	cursor := strings.TrimSpace(req.Cursor)
	if cursor == "" {
		cursor = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	backfill := strings.TrimSpace(req.Cursor) != ""

	// Missing creds should still allow deterministic smoke/demo behavior.
	demoMode := req.Demo || v.baseURL == "" || v.token == "" || strings.Contains(strings.ToLower(v.baseURL), "example")

	var (
		records []uiSPDeviceRecord
		err     error
	)
	if demoMode {
		records = v.demoRecords()
	} else {
		records, err = v.fetchVendorRecords(ctx, req.Retries)
		if err != nil {
			v.setStatus(SourceStatus{
				Source:     v.source,
				LastPollAt: time.Now().UTC().Format(time.RFC3339),
				LastCursor: cursor,
				LastError:  err.Error(),
				Demo:       false,
				Stub:       true,
			})
			return sourcePollBatch{
				Response: SourcePollResponse{
					Source:     v.source,
					Cursor:     cursor,
					Backfill:   backfill,
					Demo:       false,
					DurationMs: time.Since(start).Milliseconds(),
					Stub:       true,
					Error:      err.Error(),
				},
			}, err
		}
	}
	if len(records) > req.Limit {
		records = records[:req.Limit]
	}

	nowMs := time.Now().UnixMilli()
	events := make([]TelemetryIngestRequest, 0, len(records))
	normalized := 0
	deduped := 0
	emitted := 0

	v.mu.Lock()
	for key, ts := range v.seen {
		if nowMs-ts > int64((1 * time.Hour).Milliseconds()) {
			delete(v.seen, key)
		}
	}

	for _, rec := range records {
		if rec.ID == "" {
			continue
		}
		normalized++

		prev, seen := v.lastKnown[rec.ID]
		v.lastKnown[rec.ID] = rec.Online

		eventType := ""
		if !seen {
			if !rec.Online {
				eventType = "device_down"
			}
		} else if prev != rec.Online {
			if rec.Online {
				eventType = "device_up"
			} else {
				eventType = "device_down"
			}
		}
		if eventType == "" {
			continue
		}

		eventKey := fmt.Sprintf("%s|%s|%s", rec.ID, eventType, cursor)
		if _, ok := v.seen[eventKey]; ok {
			deduped++
			continue
		}
		v.seen[eventKey] = nowMs

		online := rec.Online
		events = append(events, TelemetryIngestRequest{
			Source:       v.source,
			EventType:    eventType,
			ObservedAtMs: rec.ObservedAtMs,
			DeviceID:     rec.ID,
			Device:       rec.Name,
			Hostname:     rec.Host,
			Mac:          rec.Mac,
			Serial:       rec.Serial,
			Model:        rec.Model,
			Vendor:       rec.Vendor,
			Role:         rec.Role,
			SiteID:       rec.SiteID,
			Online:       &online,
			LatencyMs:    rec.Latency,
			Message:      fmt.Sprintf("%s poll state=%t", strings.ToUpper(v.source), rec.Online),
			Interfaces:   rec.Ifaces,
			Neighbors:    rec.Neighs,
		})
		emitted++
	}
	v.mu.Unlock()

	resp := SourcePollResponse{
		Source:     v.source,
		Cursor:     cursor,
		Fetched:    len(records),
		Normalized: normalized,
		Emitted:    emitted,
		Deduped:    deduped,
		Backfill:   backfill,
		Demo:       demoMode,
		DurationMs: time.Since(start).Milliseconds(),
		Stub:       true,
	}

	v.setStatus(SourceStatus{
		Source:         v.source,
		LastPollAt:     time.Now().UTC().Format(time.RFC3339),
		LastCursor:     cursor,
		LastFetched:    resp.Fetched,
		LastNormalized: resp.Normalized,
		LastEmitted:    resp.Emitted,
		Demo:           demoMode,
		Stub:           true,
	})

	return sourcePollBatch{Response: resp, Events: events}, nil
}

func (v *VendorConnector) fetchVendorRecords(ctx context.Context, retries int) ([]uiSPDeviceRecord, error) {
	url := v.baseURL + v.devicesPath
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		v.applyAuthHeaders(req)

		resp, err := v.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			break
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			break
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("%s status %d", v.source, resp.StatusCode)
			if attempt < retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			break
		}

		recs, parseErr := parseVendorDevices(body, v.source, v.vendorLabel)
		if parseErr != nil {
			lastErr = parseErr
			if attempt < retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			break
		}
		return recs, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%s poll failed", v.source)
	}
	return nil, lastErr
}

func (v *VendorConnector) applyAuthHeaders(req *http.Request) {
	if strings.TrimSpace(v.token) == "" {
		return
	}
	switch v.authScheme {
	case "x-auth-token":
		req.Header.Set("X-Auth-Token", v.token)
	case "x-cisco-meraki-api-key":
		req.Header.Set("X-Cisco-Meraki-API-Key", v.token)
	case "token":
		req.Header.Set("Token", v.token)
	case "authorization":
		req.Header.Set("Authorization", v.token)
	case "none":
		return
	default:
		req.Header.Set("Authorization", "Bearer "+v.token)
	}
}

func parseVendorDevices(body []byte, source, vendorLabel string) ([]uiSPDeviceRecord, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	items := extractVendorItems(payload)
	if len(items) == 0 {
		return nil, fmt.Errorf("%s response had no devices", source)
	}

	records := make([]uiSPDeviceRecord, 0, len(items))
	for _, item := range items {
		id := pickString(item,
			[]string{"id"},
			[]string{"deviceId"},
			[]string{"device_id"},
			[]string{"serial"},
			[]string{"serialNumber"},
			[]string{"mac"},
			[]string{"hostname"},
			[]string{"name"},
		)
		if strings.TrimSpace(id) == "" {
			continue
		}

		name := pickString(item,
			[]string{"name"},
			[]string{"displayName"},
			[]string{"deviceName"},
			[]string{"hostname"},
			[]string{"hostName"},
			[]string{"serial"},
		)
		if name == "" {
			name = id
		}

		role := pickString(item,
			[]string{"role"},
			[]string{"type"},
			[]string{"deviceType"},
			[]string{"productType"},
			[]string{"category"},
		)
		if role == "" {
			role = "device"
		}
		role = normalizeVendorRole(role)

		siteID := pickString(item,
			[]string{"siteId"},
			[]string{"site_id"},
			[]string{"site", "id"},
			[]string{"networkId"},
			[]string{"network_id"},
			[]string{"location", "id"},
		)
		if siteID == "" {
			siteID = source
		}

		online := true
		if boolVal := pickBool(item,
			[]string{"online"},
			[]string{"isOnline"},
			[]string{"reachable"},
			[]string{"connected"},
			[]string{"status", "online"},
			[]string{"status", "reachable"},
		); boolVal != nil {
			online = *boolVal
		} else {
			state := strings.ToLower(strings.TrimSpace(pickString(item,
				[]string{"status"},
				[]string{"state"},
				[]string{"health"},
				[]string{"connectionState"},
				[]string{"connectivity"},
			)))
			if state != "" {
				online = parseConnectorOnlineState(state)
			}
		}

		latency := pickFloat(item,
			[]string{"latency"},
			[]string{"latencyMs"},
			[]string{"latency_ms"},
			[]string{"ping"},
			[]string{"status", "latencyMs"},
		)
		observedAtMs := pickTimestampMs(item,
			[]string{"lastSeen"},
			[]string{"lastSeenMs"},
			[]string{"last_seen"},
			[]string{"updatedAt"},
			[]string{"timestamp"},
		)

		vendor := pickString(item,
			[]string{"vendor"},
			[]string{"manufacturer"},
		)
		if vendor == "" {
			vendor = vendorLabel
		}

		records = append(records, uiSPDeviceRecord{
			ID:     id,
			Name:   name,
			Role:   role,
			SiteID: siteID,
			Host: pickString(item,
				[]string{"hostname"},
				[]string{"hostName"},
				[]string{"host"},
			),
			Mac: pickString(item,
				[]string{"mac"},
				[]string{"macAddress"},
				[]string{"mac_address"},
			),
			Serial: pickString(item,
				[]string{"serial"},
				[]string{"serialNumber"},
			),
			Model: pickString(item,
				[]string{"model"},
				[]string{"product"},
				[]string{"sku"},
			),
			Vendor:       vendor,
			Ifaces:       parseUISPInterfaces(item),
			Neighs:       parseUISPNeighbors(item),
			Online:       online,
			Latency:      latency,
			ObservedAtMs: observedAtMs,
		})
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%s response had no valid records", source)
	}
	return records, nil
}

func extractVendorItems(payload any) []map[string]any {
	items := make([]map[string]any, 0)
	switch v := payload.(type) {
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				items = append(items, m)
			}
		}
	case map[string]any:
		for _, key := range []string{"devices", "items", "data", "results", "nodes"} {
			if arr, ok := v[key].([]any); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						items = append(items, m)
					}
				}
				if len(items) > 0 {
					return items
				}
			}
		}
	}
	return items
}

func parseConnectorOnlineState(state string) bool {
	state = strings.TrimSpace(strings.ToLower(state))
	if state == "" {
		return true
	}
	switch state {
	case "online", "up", "connected", "active", "ok", "healthy", "reachable", "ready", "enabled", "alerting":
		return true
	case "offline", "down", "disconnected", "inactive", "failed", "degraded", "unreachable", "disabled", "critical", "dormant":
		return false
	default:
		return true
	}
}

func normalizeVendorRole(role string) string {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		return "router"
	}
	switch {
	case strings.Contains(role, "gateway"),
		strings.Contains(role, "firewall"),
		strings.Contains(role, "edge"),
		strings.Contains(role, "appliance"),
		strings.Contains(role, "cellulargateway"),
		strings.Contains(role, "campusgateway"),
		strings.Contains(role, "secureconnect"):
		return "gateway"
	case strings.Contains(role, "switch"):
		return "switch"
	case strings.Contains(role, "ap"),
		strings.Contains(role, "wireless"),
		strings.Contains(role, "access point"):
		return "ap"
	case strings.Contains(role, "router"):
		return "router"
	default:
		return "router"
	}
}

func (v *VendorConnector) demoRecords() []uiSPDeviceRecord {
	latFast := 6.0
	latWarn := 160.0
	tick := time.Now().Unix() / 30
	flapOnline := tick%2 == 0
	prefix := strings.TrimSpace(v.source)
	if prefix == "" {
		prefix = "vendor"
	}
	vendor := strings.TrimSpace(v.vendorLabel)
	if vendor == "" {
		vendor = connectorLabelFromSource(prefix)
	}

	return []uiSPDeviceRecord{
		{
			ID:      prefix + "-core-1",
			Name:    strings.ToUpper(prefix) + " Core 1",
			Role:    "gateway",
			SiteID:  "site-demo",
			Online:  true,
			Latency: &latFast,
			Vendor:  vendor,
		},
		{
			ID:      prefix + "-edge-1",
			Name:    strings.ToUpper(prefix) + " Edge 1",
			Role:    "switch",
			SiteID:  "site-demo",
			Online:  flapOnline,
			Latency: &latWarn,
			Vendor:  vendor,
		},
		{
			ID:      prefix + "-ap-1",
			Name:    strings.ToUpper(prefix) + " AP 1",
			Role:    "ap",
			SiteID:  "site-demo",
			Online:  true,
			Latency: &latFast,
			Vendor:  vendor,
		},
	}
}

func (v *VendorConnector) setStatus(status SourceStatus) {
	v.mu.Lock()
	v.status = status
	v.mu.Unlock()
}

func connectorLabelFromSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "Vendor"
	}
	runes := []rune(source)
	if len(runes) == 0 {
		return "Vendor"
	}
	first := strings.ToUpper(string(runes[0]))
	if len(runes) == 1 {
		return first
	}
	return first + strings.ToLower(string(runes[1:]))
}
