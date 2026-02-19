package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SourceConnector interface {
	Name() string
	Poll(ctx context.Context, req SourcePollRequest) (sourcePollBatch, error)
	Status() SourceStatus
}

type sourcePollBatch struct {
	Response SourcePollResponse
	Events   []TelemetryIngestRequest
}

type uiSPDeviceRecord struct {
	ID      string
	Name    string
	Role    string
	SiteID  string
	Host    string
	Mac     string
	Serial  string
	Model   string
	Vendor  string
	Online  bool
	Latency *float64
}

type UISPConnector struct {
	baseURL     string
	token       string
	devicesPath string
	client      *http.Client

	mu        sync.RWMutex
	status    SourceStatus
	lastKnown map[string]bool
	seen      map[string]int64
}

func NewUISPConnector(baseURL, token, devicesPath string) *UISPConnector {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if devicesPath == "" {
		devicesPath = "/nms/api/v2.1/devices"
	}
	if !strings.HasPrefix(devicesPath, "/") {
		devicesPath = "/" + devicesPath
	}
	return &UISPConnector{
		baseURL:     baseURL,
		token:       strings.TrimSpace(token),
		devicesPath: devicesPath,
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
		status:    SourceStatus{Source: "uisp", Stub: true},
		lastKnown: map[string]bool{},
		seen:      map[string]int64{},
	}
}

func (u *UISPConnector) Name() string {
	return "uisp"
}

func (u *UISPConnector) Status() SourceStatus {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.status
}

func (u *UISPConnector) Poll(ctx context.Context, req SourcePollRequest) (sourcePollBatch, error) {
	start := time.Now()
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 200
	}
	if req.Retries < 0 {
		req.Retries = 0
	}
	cursor := strings.TrimSpace(req.Cursor)
	if cursor == "" {
		cursor = strconv.FormatInt(time.Now().UnixMilli(), 10)
	}

	demoMode := req.Demo || u.baseURL == "" || u.token == "" || strings.Contains(strings.ToLower(u.baseURL), "example")
	backfill := strings.TrimSpace(req.Cursor) != ""

	var (
		records []uiSPDeviceRecord
		err     error
	)
	if demoMode {
		records = u.demoRecords()
	} else {
		records, err = u.fetchUISPRecords(ctx, req.Retries)
		if err != nil {
			u.setStatus(SourceStatus{
				Source:     "uisp",
				LastPollAt: time.Now().UTC().Format(time.RFC3339),
				LastCursor: cursor,
				LastError:  err.Error(),
				Demo:       false,
				Stub:       true,
			})
			return sourcePollBatch{
				Response: SourcePollResponse{
					Source:     "uisp",
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

	u.mu.Lock()
	for id, ts := range u.seen {
		if nowMs-ts > int64((1 * time.Hour).Milliseconds()) {
			delete(u.seen, id)
		}
	}

	for _, rec := range records {
		if rec.ID == "" {
			continue
		}
		normalized++

		prev, seen := u.lastKnown[rec.ID]
		u.lastKnown[rec.ID] = rec.Online

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
		if _, ok := u.seen[eventKey]; ok {
			deduped++
			continue
		}
		u.seen[eventKey] = nowMs

		online := rec.Online
		events = append(events, TelemetryIngestRequest{
			Source:    "uisp",
			EventType: eventType,
			DeviceID:  rec.ID,
			Device:    rec.Name,
			Hostname:  rec.Host,
			Mac:       rec.Mac,
			Serial:    rec.Serial,
			Model:     rec.Model,
			Vendor:    rec.Vendor,
			Role:      rec.Role,
			SiteID:    rec.SiteID,
			Online:    &online,
			LatencyMs: rec.Latency,
			Message:   fmt.Sprintf("UISP poll state=%t", rec.Online),
		})
		emitted++
	}
	u.mu.Unlock()

	resp := SourcePollResponse{
		Source:     "uisp",
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

	u.setStatus(SourceStatus{
		Source:         "uisp",
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

func (u *UISPConnector) fetchUISPRecords(ctx context.Context, retries int) ([]uiSPDeviceRecord, error) {
	url := u.baseURL + u.devicesPath
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Auth-Token", u.token)
		req.Header.Set("Authorization", "Bearer "+u.token)

		resp, err := u.client.Do(req)
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
			lastErr = fmt.Errorf("uisp status %d", resp.StatusCode)
			if attempt < retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
				continue
			}
			break
		}

		recs, parseErr := parseUISPDevices(body)
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
		lastErr = fmt.Errorf("uisp poll failed")
	}
	return nil, lastErr
}
func parseUISPDevices(body []byte) ([]uiSPDeviceRecord, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0)
	switch v := payload.(type) {
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				items = append(items, m)
			}
		}
	case map[string]any:
		if arr, ok := v["devices"].([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("uisp response had no devices")
	}

	records := make([]uiSPDeviceRecord, 0, len(items))
	for _, item := range items {
		id := pickString(item,
			[]string{"identification", "id"},
			[]string{"identification", "mac"},
			[]string{"id"},
		)
		if id == "" {
			continue
		}
		name := pickString(item,
			[]string{"identification", "name"},
			[]string{"identification", "displayName"},
			[]string{"name"},
		)
		if name == "" {
			name = id
		}
		role := pickString(item,
			[]string{"identification", "role"},
			[]string{"role"},
		)
		if role == "" {
			role = "device"
		}
		siteID := pickString(item,
			[]string{"site", "id"},
			[]string{"siteId"},
		)
		if siteID == "" {
			siteID = "uisp"
		}

		status := strings.ToLower(strings.TrimSpace(pickString(item,
			[]string{"overview", "status"},
			[]string{"status"},
		)))
		online := status == "ok" || status == "online" || status == "active" || status == "connected" || status == "reachable" || status == "enabled"

		latency := pickFloat(item,
			[]string{"overview", "latency"},
			[]string{"overview", "ping"},
		)

		records = append(records, uiSPDeviceRecord{
			ID:     id,
			Name:   name,
			Role:   role,
			SiteID: siteID,
			Host: pickString(item,
				[]string{"identification", "hostname"},
				[]string{"hostname"},
			),
			Mac: pickString(item,
				[]string{"identification", "mac"},
				[]string{"mac"},
			),
			Serial: pickString(item,
				[]string{"identification", "serialNumber"},
				[]string{"identification", "serial"},
				[]string{"serialNumber"},
				[]string{"serial"},
			),
			Model: pickString(item,
				[]string{"identification", "model"},
				[]string{"model"},
			),
			Vendor: pickString(item,
				[]string{"identification", "vendor"},
				[]string{"vendor"},
			),
			Online:  online,
			Latency: latency,
		})
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("uisp response had no valid records")
	}
	return records, nil
}

func pickString(m map[string]any, paths ...[]string) string {
	for _, p := range paths {
		if v, ok := nestedValue(m, p...); ok {
			switch t := v.(type) {
			case string:
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

func pickFloat(m map[string]any, paths ...[]string) *float64 {
	for _, p := range paths {
		if v, ok := nestedValue(m, p...); ok {
			switch t := v.(type) {
			case float64:
				val := t
				return &val
			case int:
				val := float64(t)
				return &val
			case string:
				if parsed, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
					val := parsed
					return &val
				}
			}
		}
	}
	return nil
}

func nestedValue(m map[string]any, keys ...string) (any, bool) {
	var current any = m
	for _, k := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := asMap[k]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func (u *UISPConnector) demoRecords() []uiSPDeviceRecord {
	latFast := 3.0
	latWarn := 180.0
	tick := time.Now().Unix() / 30
	apOnline := tick%2 == 0
	return []uiSPDeviceRecord{
		{ID: "uisp-gw-1", Name: "UISP Gateway 1", Role: "gateway", SiteID: "site-demo", Online: true, Latency: &latFast},
		{ID: "uisp-ap-1", Name: "UISP AP 1", Role: "ap", SiteID: "site-demo", Online: apOnline, Latency: &latWarn},
		{ID: "uisp-sw-1", Name: "UISP Switch 1", Role: "switch", SiteID: "site-demo", Online: true, Latency: &latFast},
	}
}

func (u *UISPConnector) setStatus(status SourceStatus) {
	u.mu.Lock()
	u.status = status
	u.mu.Unlock()
}

func ingestSourceEvents(store *Store, events []TelemetryIngestRequest) (int, int) {
	ingested := 0
	incidents := 0
	for _, ev := range events {
		_, inc, ok := store.IngestTelemetry(ev)
		if !ok {
			continue
		}
		ingested++
		if inc != nil {
			incidents++
		}
	}
	return ingested, incidents
}

func runSourcePoller(ctx context.Context, connector SourceConnector, store *Store, logger *slog.Logger, interval time.Duration, retries int) {
	if interval <= 0 {
		return
	}
	logger.Info("source_poller_started", "source", connector.Name(), "interval_sec", int(interval.Seconds()), "retries", retries)

	run := func() {
		batch, err := connector.Poll(ctx, SourcePollRequest{Retries: retries})
		if err != nil {
			logger.Warn("source_poller_poll_failed", "source", connector.Name(), "error", err.Error())
			return
		}
		ingested, incidents := ingestSourceEvents(store, batch.Events)
		logger.Info("source_poller_poll_ok",
			"source", connector.Name(),
			"fetched", batch.Response.Fetched,
			"normalized", batch.Response.Normalized,
			"emitted", batch.Response.Emitted,
			"ingested", ingested,
			"incidents", incidents,
			"demo", batch.Response.Demo,
		)
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("source_poller_stopped", "source", connector.Name())
			return
		case <-ticker.C:
			run()
		}
	}
}
