package main

import "testing"

func TestIngestSourceEventsUsesPriorityQueueOrder(t *testing.T) {
	s := LoadStore("")

	s.mu.Lock()
	s.TelemetryGovernorRules = normalizeTelemetryGovernorRules([]TelemetryClassGovernorRule{
		{
			DeviceClass:         "core",
			MinSampleIntervalMs: 1000,
			QueuePriority:       0,
			Roles:               []string{"gateway"},
		},
		{
			DeviceClass:         "access",
			MinSampleIntervalMs: 1000,
			QueuePriority:       2,
			Roles:               []string{"switch"},
		},
		{
			DeviceClass:         "edge",
			MinSampleIntervalMs: 1000,
			QueuePriority:       4,
			Roles:               []string{"station"},
		},
		{
			DeviceClass:         "default",
			MinSampleIntervalMs: 1000,
			QueuePriority:       9,
			Roles:               []string{"device"},
		},
	})
	s.TelemetryHot = nil
	s.TelemetryWarm = nil
	s.TelemetryCold = nil
	s.mu.Unlock()

	online := true
	ingested, incidents, dropped := ingestSourceEvents(s, []TelemetryIngestRequest{
		{Source: "queue_test", DeviceID: "edge-1", Device: "edge-1", Role: "station", Online: &online},
		{Source: "queue_test", DeviceID: "core-1", Device: "core-1", Role: "gateway", Online: &online},
		{Source: "queue_test", DeviceID: "access-1", Device: "access-1", Role: "switch", Online: &online},
	})
	if ingested != 3 || incidents != 0 || dropped != 0 {
		t.Fatalf("expected ingest stats 3/0/0, got %d/%d/%d", ingested, incidents, dropped)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.TelemetryHot) != 3 {
		t.Fatalf("expected 3 telemetry hot samples, got=%d", len(s.TelemetryHot))
	}
	if s.TelemetryHot[0].DeviceID != "core-1" || s.TelemetryHot[1].DeviceID != "access-1" || s.TelemetryHot[2].DeviceID != "edge-1" {
		t.Fatalf("unexpected ingest order in telemetry hot tier: %#v", s.TelemetryHot)
	}
}
