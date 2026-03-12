package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

	uispConnector := NewUISPConnector(
		getenv("UISP_URL", ""),
		getenv("UISP_TOKEN", ""),
		getenv("UISP_DEVICES_PATH", "/nms/api/v2.1/devices"),
	)
	ciscoConnector := NewVendorConnector(
		"cisco",
		"Cisco",
		getenv("CISCO_URL", ""),
		getenv("CISCO_TOKEN", ""),
		getenv("CISCO_DEVICES_PATH", "/api/v1/devices"),
		getenv("CISCO_AUTH_SCHEME", "bearer"),
	)
	juniperConnector := NewVendorConnector(
		"juniper",
		"Juniper",
		getenv("JUNIPER_URL", ""),
		getenv("JUNIPER_TOKEN", ""),
		getenv("JUNIPER_DEVICES_PATH", "/api/v1/devices"),
		getenv("JUNIPER_AUTH_SCHEME", "bearer"),
	)
	merakiConnector := NewVendorConnector(
		"meraki",
		"Meraki",
		getenv("MERAKI_URL", ""),
		getenv("MERAKI_TOKEN", ""),
		getenv("MERAKI_DEVICES_PATH", "/devices/statuses"),
		getenv("MERAKI_AUTH_SCHEME", "x-cisco-meraki-api-key"),
	)

	pollSec := getenvInt("UISP_POLL_INTERVAL_SEC", 0)
	pollRetries := getenvInt("UISP_POLL_RETRIES", 1)
	if pollSec > 0 {
		go runSourcePoller(context.Background(), uispConnector, store, logger, time.Duration(pollSec)*time.Second, pollRetries)
	}
	ciscoPollSec := getenvInt("CISCO_POLL_INTERVAL_SEC", 0)
	ciscoPollRetries := getenvInt("CISCO_POLL_RETRIES", 1)
	if ciscoPollSec > 0 {
		go runSourcePoller(context.Background(), ciscoConnector, store, logger, time.Duration(ciscoPollSec)*time.Second, ciscoPollRetries)
	}
	juniperPollSec := getenvInt("JUNIPER_POLL_INTERVAL_SEC", 0)
	juniperPollRetries := getenvInt("JUNIPER_POLL_RETRIES", 1)
	if juniperPollSec > 0 {
		go runSourcePoller(context.Background(), juniperConnector, store, logger, time.Duration(juniperPollSec)*time.Second, juniperPollRetries)
	}
	merakiPollSec := getenvInt("MERAKI_POLL_INTERVAL_SEC", 0)
	merakiPollRetries := getenvInt("MERAKI_POLL_RETRIES", 1)
	if merakiPollSec > 0 {
		go runSourcePoller(context.Background(), merakiConnector, store, logger, time.Duration(merakiPollSec)*time.Second, merakiPollRetries)
	}

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
				"native_api":                   true,
				"agent_ingest":                 true,
				"events_ingest":                true,
				"source_uisp_poll":             true,
				"source_cisco_poll":            true,
				"source_juniper_poll":          true,
				"source_meraki_poll":           true,
				"topology_api":                 true,
				"topology_path_trace":          true,
				"topology_ha_watcher":          true,
				"telemetry_sampling_governor":  true,
				"telemetry_gap_detector":       true,
				"telemetry_quality_scorecards": true,
				"telemetry_ingestion_health":   true,
				"telemetry_dynamic_baseline":   true,
				"telemetry_anomaly_windows":    true,
				"telemetry_alert_confidence":   true,
				"telemetry_impact_radius":      true,
				"telemetry_storm_shield":       true,
				"telemetry_alert_comparison":   true,
				"incident_commander_mode":      true,
				"incident_command_timeline":    true,
				"incident_workspace_mode":      true,
				"incident_shift_handoff":       true,
				"incident_audit_events":        true,
				"source_poll_background":       pollSec > 0 || ciscoPollSec > 0 || juniperPollSec > 0 || merakiPollSec > 0,
				"cloud_multi_tenant_stub":      true,
				"connector_multivendor_stub":   false,
			},
			PushRegister: apiBase + "/push/register",
			Environment:  getenv("APP_ENV", "dev"),
			Version:      "0.3.0",
			Banner:       "NOCWALL-CE API (UISP source poller stub)",
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

	app.Get("/incidents/workspace", authMiddleware, func(c *fiber.Ctx) error {
		activeLimit := c.QueryInt("active_limit", 80)
		recentLimit := c.QueryInt("recent_limit", 40)
		return c.JSON(store.IncidentWorkspace(activeLimit, recentLimit))
	})

	app.Get("/incidents/handoffs", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 30)
		handoffs, truncated, normalizedLimit := store.ListIncidentHandoffs(limit)
		return c.JSON(IncidentHandoffHistoryResponse{
			LastUpdatedMs: time.Now().UnixMilli(),
			Count:         len(handoffs),
			Handoffs:      handoffs,
			Truncated:     truncated,
			Limit:         normalizedLimit,
			Stub:          true,
		})
	})

	app.Post("/incidents/handoff/generate", authMiddleware, func(c *fiber.Ctx) error {
		var req IncidentHandoffGenerateRequest
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&req); err != nil {
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
			}
		}
		return c.JSON(store.GenerateIncidentShiftHandoff(req.Actor, req.Note, req.ActiveLimit))
	})

	app.Get("/incidents/:id/export", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		format := normalizeIncidentExportFormat(c.Query("format", "markdown"))
		if format == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_format", "message": "format must be markdown or pdf"})
		}

		doc, ok := store.IncidentTimelineExport(id)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
		}

		filename := IncidentTimelineExportFilename(doc.Incident, format)
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		switch format {
		case "pdf":
			c.Type("pdf")
			return c.Send(BuildIncidentTimelinePDF(doc))
		default:
			c.Set(fiber.HeaderContentType, "text/markdown; charset=utf-8")
			return c.SendString(BuildIncidentTimelineMarkdown(doc))
		}
	})

	app.Get("/incidents/audit", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 120)
		incidentID := strings.TrimSpace(c.Query("incident_id", ""))
		action := strings.TrimSpace(c.Query("action", ""))
		events, truncated, normalizedLimit := store.ListIncidentAuditEvents(limit, incidentID, action)
		return c.JSON(IncidentAuditEventsResponse{
			LastUpdatedMs: time.Now().UnixMilli(),
			Count:         len(events),
			Events:        events,
			Truncated:     truncated,
			Limit:         normalizedLimit,
			Stub:          true,
		})
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

	app.Post("/incidents/:id/commander", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var req IncidentCommanderRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		inc, ok := store.SetIncidentCommander(id, req.Commander, req.Actor)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
		}
		return c.JSON(inc)
	})

	app.Post("/incidents/:id/timeline", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var req IncidentTimelineRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		if strings.TrimSpace(req.Message) == "" {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "missing_message", "message": "message is required"})
		}
		inc, ok := store.AddIncidentTimelineEntry(id, req.EventType, req.Message, req.Actor)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
		}
		return c.JSON(inc)
	})

	app.Post("/incidents/:id/checklist/audit", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		var req IncidentChecklistAuditRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		event, ok := store.RecordIncidentChecklistAction(id, req.ChecklistID, req.StepID, req.State, req.Actor, req.Note)
		if !ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": "not_found", "message": "Incident not found"})
		}
		return c.JSON(event)
	})

	app.Get("/metrics/devices/:id", authMiddleware, func(c *fiber.Ctx) error {
		id := c.Params("id")
		points := []fiber.Map{
			{"timestamp": time.Now().Add(-5 * time.Minute).Unix(), "latency": 5, "cpu": 20, "ram": 30, "online": true},
			{"timestamp": time.Now().Unix(), "latency": 8, "cpu": 22, "ram": 31, "online": true},
		}
		return c.JSON(fiber.Map{"device_id": id, "points": points})
	})

	app.Get("/telemetry/retention", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.LastRetentionSummary())
	})

	app.Get("/telemetry/governor", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.TelemetryGovernorStatus())
	})

	app.Get("/telemetry/quality", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.TelemetryQualityReport())
	})

	app.Get("/telemetry/ingestion/health", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.TelemetryIngestionHealth())
	})

	app.Get("/telemetry/baselines", authMiddleware, func(c *fiber.Ctx) error {
		windowHours := c.QueryInt("window_hours", defaultBaselineHours)
		return c.JSON(store.TelemetryBaselineReport(windowHours))
	})

	app.Get("/telemetry/alerts/intelligence", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 40)
		windowMinutes := c.QueryInt("window_minutes", defaultAlertWindowMins)
		burstThreshold := c.QueryInt("burst_threshold", defaultBurstThreshold)
		return c.JSON(store.TelemetryAlertIntelligence(limit, windowMinutes, burstThreshold))
	})

	registerSourceRoutes := func(source string, connector SourceConnector) {
		app.Get("/sources/"+source+"/status", authMiddleware, func(c *fiber.Ctx) error {
			return c.JSON(connector.Status())
		})
		app.Post("/sources/"+source+"/poll", authMiddleware, func(c *fiber.Ctx) error {
			var req SourcePollRequest
			if len(c.Body()) > 0 {
				if err := c.BodyParser(&req); err != nil {
					return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
				}
			}

			batch, err := connector.Poll(c.Context(), req)
			if err != nil {
				store.RecordSourcePollOutcome(source, false, err.Error(), time.Now().UnixMilli())
				store.DetectTelemetryGaps(time.Now().UnixMilli())
				resp := batch.Response
				resp.Stub = true
				return c.Status(http.StatusBadGateway).JSON(resp)
			}
			store.RecordSourcePollOutcome(source, true, "", time.Now().UnixMilli())
			ingested, incidents, dropped := ingestSourceEvents(store, batch.Events)
			gapsCreated, gapsResolved := store.DetectTelemetryGaps(time.Now().UnixMilli())
			batch.Response.Ingested = ingested
			batch.Response.DroppedByGovernor = dropped
			batch.Response.IncidentsCreated = incidents
			batch.Response.Stub = true
			logger.Info("source_poll_manual",
				"source", source,
				"fetched", batch.Response.Fetched,
				"normalized", batch.Response.Normalized,
				"emitted", batch.Response.Emitted,
				"ingested", ingested,
				"dropped_by_governor", dropped,
				"incidents", incidents,
				"gap_incidents_created", gapsCreated,
				"gap_incidents_resolved", gapsResolved,
				"demo", batch.Response.Demo,
			)
			return c.JSON(batch.Response)
		})
	}
	registerSourceRoutes("uisp", uispConnector)
	registerSourceRoutes("cisco", ciscoConnector)
	registerSourceRoutes("juniper", juniperConnector)
	registerSourceRoutes("meraki", merakiConnector)

	app.Get("/inventory/schema", authMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(store.InventorySchema())
	})

	app.Get("/inventory/identities", authMiddleware, func(c *fiber.Ctx) error {
		identities := store.ListDeviceIdentities()
		return c.JSON(InventoryIdentitiesResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(identities),
			Identities:  identities,
			Stub:        true,
		})
	})

	app.Get("/inventory/observations", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		identityID := c.Query("identity_id", "")
		observations, truncated, normalizedLimit := store.ListSourceObservations(limit, identityID)
		return c.JSON(InventoryObservationsResponse{
			LastUpdated:  time.Now().UnixMilli(),
			Count:        len(observations),
			Observations: observations,
			Truncated:    truncated,
			Limit:        normalizedLimit,
			Stub:         true,
		})
	})

	app.Get("/inventory/drift", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		identityID := c.Query("identity_id", "")
		snapshots, truncated, normalizedLimit := store.ListDriftSnapshots(limit, identityID)
		return c.JSON(InventoryDriftResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(snapshots),
			Snapshots:   snapshots,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/inventory/interfaces", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		identityID := c.Query("identity_id", "")
		items, truncated, normalizedLimit := store.ListDeviceInterfaces(limit, identityID)
		return c.JSON(InventoryInterfacesResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(items),
			Interfaces:  items,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/inventory/neighbors", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		identityID := c.Query("identity_id", "")
		items, truncated, normalizedLimit := store.ListNeighborLinks(limit, identityID)
		return c.JSON(InventoryNeighborsResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(items),
			Neighbors:   items,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/inventory/lifecycle", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		identityID := c.Query("identity_id", "")
		items, truncated, normalizedLimit := store.ListLifecycleScores(limit, identityID)
		return c.JSON(InventoryLifecycleResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(items),
			Scores:      items,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/topology/nodes", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 300)
		siteID := c.Query("site_id", "")
		items, truncated, normalizedLimit := store.ListTopologyNodes(limit, siteID)
		return c.JSON(TopologyNodesResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(items),
			Nodes:       items,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/topology/edges", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 300)
		identityID := c.Query("identity_id", "")
		items, truncated, normalizedLimit := store.ListTopologyEdges(limit, identityID)
		return c.JSON(TopologyEdgesResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(items),
			Edges:       items,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/topology/health", authMiddleware, func(c *fiber.Ctx) error {
		health := store.TopologyHealth()
		return c.JSON(TopologyHealthResponse{
			LastUpdated: time.Now().UnixMilli(),
			Health:      health,
			Stub:        true,
		})
	})

	app.Get("/topology/ha/pairs", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		state := strings.TrimSpace(c.Query("state", ""))
		pairs, truncated, normalizedLimit := store.ListHAPairs(limit, state)
		return c.JSON(TopologyHAPairsResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(pairs),
			Pairs:       pairs,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/topology/ha/events", authMiddleware, func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 200)
		pairID := strings.TrimSpace(c.Query("pair_id", ""))
		eventType := strings.TrimSpace(c.Query("event_type", ""))
		events, truncated, normalizedLimit := store.ListHAFailoverEvents(limit, pairID, eventType)
		return c.JSON(TopologyHAEventsResponse{
			LastUpdated: time.Now().UnixMilli(),
			Count:       len(events),
			Events:      events,
			Truncated:   truncated,
			Limit:       normalizedLimit,
			Stub:        true,
		})
	})

	app.Get("/topology/path", authMiddleware, func(c *fiber.Ctx) error {
		sourceIdentityID := strings.TrimSpace(c.Query("source_identity_id", ""))
		targetIdentityID := strings.TrimSpace(c.Query("target_identity_id", ""))
		sourceNodeID := strings.TrimSpace(c.Query("source_node_id", ""))
		targetNodeID := strings.TrimSpace(c.Query("target_node_id", ""))

		nodes, edges, found, message := store.TraceTopologyPath(sourceIdentityID, targetIdentityID, sourceNodeID, targetNodeID)
		return c.JSON(TopologyPathResponse{
			LastUpdated:      time.Now().UnixMilli(),
			Found:            found,
			SourceNodeID:     sourceNodeID,
			TargetNodeID:     targetNodeID,
			SourceIdentityID: sourceIdentityID,
			TargetIdentityID: targetIdentityID,
			Hops:             max(0, len(nodes)-1),
			Nodes:            nodes,
			Edges:            edges,
			Message:          message,
			Stub:             true,
		})
	})

	app.Post("/inventory/identities/merge", authMiddleware, func(c *fiber.Ctx) error {
		var req IdentityMergeRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": "invalid_body", "message": "Invalid request body"})
		}
		secondary := make([]string, 0, len(req.SecondaryIDs)+1)
		if strings.TrimSpace(req.SecondaryID) != "" {
			secondary = append(secondary, req.SecondaryID)
		}
		secondary = append(secondary, req.SecondaryIDs...)

		primary, merged, err := store.MergeIdentities(req.PrimaryID, secondary)
		if err != nil {
			switch err {
			case ErrInvalidPrimary, ErrNoSecondary:
				return c.Status(http.StatusBadRequest).JSON(fiber.Map{"code": err.Error(), "message": err.Error()})
			case ErrPrimaryNotFound:
				return c.Status(http.StatusNotFound).JSON(fiber.Map{"code": err.Error(), "message": err.Error()})
			default:
				return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"code": "merge_failed", "message": err.Error()})
			}
		}

		return c.JSON(IdentityMergeResponse{
			OK:      true,
			Primary: primary,
			Merged:  merged,
			Stub:    true,
			Message: "identities merged",
		})
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
	logger.Info("api_listening", "addr", addr, "data_file", dataFile, "uisp_poll_interval_sec", pollSec)
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

func getenvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return parsed
}

func randomID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
