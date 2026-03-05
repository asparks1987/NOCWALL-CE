package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func loadTopologyFixture(t *testing.T, relativePath string) []TelemetryIngestRequest {
	t.Helper()
	path := filepath.Join("testdata", "topology", relativePath)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read topology fixture %s: %v", path, err)
	}
	var reqs []TelemetryIngestRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		t.Fatalf("decode topology fixture %s: %v", path, err)
	}
	if len(reqs) == 0 {
		t.Fatalf("topology fixture %s contained no requests", path)
	}
	return reqs
}

func ingestTelemetryBatch(t *testing.T, s *Store, reqs []TelemetryIngestRequest) {
	t.Helper()
	for i, req := range reqs {
		if req.DeviceID == "" {
			t.Fatalf("fixture request at index %d missing device_id", i)
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("fixture ingest failed index=%d device_id=%s", i, req.DeviceID)
		}
	}
}

func findIdentityByPrimary(t *testing.T, s *Store, primaryID string) DeviceIdentity {
	t.Helper()
	for _, ident := range s.ListDeviceIdentities() {
		if ident.PrimaryDeviceID == primaryID {
			return ident
		}
	}
	t.Fatalf("identity not found for primary_device_id=%s", primaryID)
	return DeviceIdentity{}
}

func containsIdentityID(items []DeviceIdentity, identityID string) bool {
	for _, ident := range items {
		if ident.IdentityID == identityID {
			return true
		}
	}
	return false
}

func telemetrySampleIDs(samples []TelemetrySample) map[string]struct{} {
	out := make(map[string]struct{}, len(samples))
	for _, sample := range samples {
		out[sample.SampleID] = struct{}{}
	}
	return out
}

func findQualityCard(cards []TelemetrySourceQualityScorecard, source string) (TelemetrySourceQualityScorecard, bool) {
	for _, card := range cards {
		if card.Source == source {
			return card, true
		}
	}
	return TelemetrySourceQualityScorecard{}, false
}

func findBaselineGroup(groups []TelemetryRoleSiteBaseline, role, siteID string) (TelemetryRoleSiteBaseline, bool) {
	for _, group := range groups {
		if group.Role == role && group.SiteID == siteID {
			return group, true
		}
	}
	return TelemetryRoleSiteBaseline{}, false
}

func findBaselineMetric(metrics []TelemetryBaselineMetric, metric string) (TelemetryBaselineMetric, bool) {
	for _, item := range metrics {
		if item.Metric == metric {
			return item, true
		}
	}
	return TelemetryBaselineMetric{}, false
}

func findAlertByDeviceID(alerts []TelemetryAlertRecord, deviceID string) (TelemetryAlertRecord, bool) {
	for _, alert := range alerts {
		if alert.Incident.DeviceID == deviceID {
			return alert, true
		}
	}
	return TelemetryAlertRecord{}, false
}

func TestIdentityGuardrailNoAutoMergeOnDistinctKeys(t *testing.T) {
	s := LoadStore("")
	reqA := TelemetryIngestRequest{
		Source:   "test_guardrail",
		DeviceID: "guard-a",
		Device:   "Guard A",
		Hostname: "guard-a.local",
		Mac:      "aa:00:00:00:00:01",
		Serial:   "SER-GA",
	}
	reqB := TelemetryIngestRequest{
		Source:   "test_guardrail",
		DeviceID: "guard-b",
		Device:   "Guard B",
		Hostname: "guard-b.local",
		Mac:      "aa:00:00:00:00:02",
		Serial:   "SER-GB",
	}

	if _, _, ok := s.IngestTelemetry(reqA); !ok {
		t.Fatalf("ingest reqA failed")
	}
	if _, _, ok := s.IngestTelemetry(reqB); !ok {
		t.Fatalf("ingest reqB failed")
	}

	identA := findIdentityByPrimary(t, s, "guard-a")
	identB := findIdentityByPrimary(t, s, "guard-b")
	if identA.IdentityID == identB.IdentityID {
		t.Fatalf("distinct-key devices merged unexpectedly: %s", identA.IdentityID)
	}
}

func TestMergeIdentitiesMovesObservationsAndInventoryFacts(t *testing.T) {
	s := LoadStore("")
	reqPrimary := TelemetryIngestRequest{
		Source:   "test_merge_primary",
		DeviceID: "merge-pri",
		Device:   "Merge Primary",
		Hostname: "merge-primary.local",
		Mac:      "bb:00:00:00:00:10",
		Serial:   "SER-MERGE-PRI",
		Interfaces: []TelemetryInterfaceFact{
			{Name: "eth0"},
		},
	}
	reqSecondary := TelemetryIngestRequest{
		Source:   "test_merge_secondary",
		DeviceID: "merge-sec",
		Device:   "Merge Secondary",
		Hostname: "merge-secondary.local",
		Mac:      "bb:00:00:00:00:20",
		Serial:   "SER-MERGE-SEC",
		Interfaces: []TelemetryInterfaceFact{
			{Name: "eth7"},
		},
		Neighbors: []TelemetryNeighborFact{
			{
				LocalInterface:       "eth7",
				NeighborIdentityHint: "edge-core-1",
				NeighborDeviceName:   "Core 1",
				NeighborInterface:    "xe-0/0/1",
				Protocol:             "lldp",
			},
		},
	}

	if _, _, ok := s.IngestTelemetry(reqPrimary); !ok {
		t.Fatalf("ingest reqPrimary failed")
	}
	if _, _, ok := s.IngestTelemetry(reqSecondary); !ok {
		t.Fatalf("ingest reqSecondary failed")
	}

	identPrimary := findIdentityByPrimary(t, s, "merge-pri")
	identSecondary := findIdentityByPrimary(t, s, "merge-sec")
	if identPrimary.IdentityID == identSecondary.IdentityID {
		t.Fatalf("precondition failed: identities already merged")
	}

	_, merged, err := s.MergeIdentities(identPrimary.IdentityID, []string{identSecondary.IdentityID})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if len(merged) != 1 || merged[0] != identSecondary.IdentityID {
		t.Fatalf("unexpected merged list: %#v", merged)
	}

	if containsIdentityID(s.ListDeviceIdentities(), identSecondary.IdentityID) {
		t.Fatalf("secondary identity still exists after merge")
	}

	ifaces, _, _ := s.ListDeviceInterfaces(200, identPrimary.IdentityID)
	if len(ifaces) < 2 {
		t.Fatalf("expected merged interfaces on primary identity, got=%d", len(ifaces))
	}
	for _, row := range ifaces {
		if row.IdentityID != identPrimary.IdentityID {
			t.Fatalf("interface not remapped to primary identity: %+v", row)
		}
	}

	neighbors, _, _ := s.ListNeighborLinks(200, identPrimary.IdentityID)
	if len(neighbors) == 0 {
		t.Fatalf("expected merged neighbor links on primary identity")
	}
	for _, row := range neighbors {
		if row.IdentityID != identPrimary.IdentityID {
			t.Fatalf("neighbor not remapped to primary identity: %+v", row)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, obs := range s.SourceObservations {
		if obs.IdentityID == identSecondary.IdentityID {
			t.Fatalf("observation still points to secondary identity")
		}
	}
}

func TestInterfaceMapperReplacesFactsPerIdentityAndSource(t *testing.T) {
	s := LoadStore("")

	reqA := TelemetryIngestRequest{
		Source:   "test_mapper",
		DeviceID: "if-source-device",
		Device:   "Mapper Device",
		Mac:      "cc:00:00:00:00:01",
		Serial:   "SER-MAPPER",
		Interfaces: []TelemetryInterfaceFact{
			{Name: "eth0"},
			{Name: "eth1"},
		},
	}
	if _, _, ok := s.IngestTelemetry(reqA); !ok {
		t.Fatalf("ingest reqA failed")
	}

	ident := findIdentityByPrimary(t, s, "if-source-device")
	ifacesA, _, _ := s.ListDeviceInterfaces(200, ident.IdentityID)
	if len(ifacesA) != 2 {
		t.Fatalf("expected 2 interfaces after first ingest, got=%d", len(ifacesA))
	}

	reqB := TelemetryIngestRequest{
		Source:   "test_mapper",
		DeviceID: "if-source-device",
		Device:   "Mapper Device",
		Mac:      "cc:00:00:00:00:01",
		Serial:   "SER-MAPPER",
		Interfaces: []TelemetryInterfaceFact{
			{Name: "eth1"},
		},
	}
	if _, _, ok := s.IngestTelemetry(reqB); !ok {
		t.Fatalf("ingest reqB failed")
	}

	ifacesB, _, _ := s.ListDeviceInterfaces(200, ident.IdentityID)
	if len(ifacesB) != 1 {
		t.Fatalf("expected source replacement to keep 1 interface, got=%d", len(ifacesB))
	}
	if ifacesB[0].Name != "eth1" {
		t.Fatalf("expected remaining interface eth1, got=%s", ifacesB[0].Name)
	}
}

func TestTelemetryRetentionIngestAddsHotSample(t *testing.T) {
	s := LoadStore("")
	s.mu.Lock()
	s.TelemetryHot = nil
	s.TelemetryWarm = nil
	s.TelemetryCold = nil
	s.mu.Unlock()

	online := true
	lat := 11.5
	req := TelemetryIngestRequest{
		Source:    "retention_test",
		DeviceID:  "ret-hot-1",
		Device:    "Retention Hot 1",
		Role:      "gateway",
		SiteID:    "ret-site",
		Online:    &online,
		LatencyMs: &lat,
	}
	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("ingest failed")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.TelemetryHot) != 1 {
		t.Fatalf("expected 1 hot sample, got=%d", len(s.TelemetryHot))
	}
	if len(s.TelemetryWarm) != 0 {
		t.Fatalf("expected 0 warm samples, got=%d", len(s.TelemetryWarm))
	}
	if len(s.TelemetryCold) != 0 {
		t.Fatalf("expected 0 cold samples, got=%d", len(s.TelemetryCold))
	}
	sample := s.TelemetryHot[0]
	if sample.DeviceID != "ret-hot-1" {
		t.Fatalf("unexpected hot sample device_id=%s", sample.DeviceID)
	}
	if sample.Source != "retention_test" {
		t.Fatalf("unexpected hot sample source=%s", sample.Source)
	}
	if sample.DeviceRole != "gateway" {
		t.Fatalf("unexpected hot sample role=%s", sample.DeviceRole)
	}
	if sample.SiteID != "ret-site" {
		t.Fatalf("unexpected hot sample site=%s", sample.SiteID)
	}
	if sample.Online == nil || !*sample.Online {
		t.Fatalf("expected hot sample online=true")
	}
	if sample.LatencyMs == nil || *sample.LatencyMs != 11.5 {
		t.Fatalf("expected hot sample latency 11.5, got=%v", sample.LatencyMs)
	}
}

func TestTelemetryRetentionPromotesAndPrunesByTierAge(t *testing.T) {
	s := LoadStore("")
	now := int64(1_700_000_010_000)
	online := true
	offline := false

	s.mu.Lock()
	s.TelemetryRetentionPolicy = TelemetryRetentionPolicy{
		HotRetentionMs:  1000,
		WarmRetentionMs: 3000,
		ColdRetentionMs: 6000,
		HotMaxSamples:   10,
		WarmMaxSamples:  10,
		ColdMaxSamples:  10,
	}
	s.TelemetryHot = []TelemetrySample{
		{SampleID: "hot-old", DeviceID: "dev-hot-old", Source: "retention_test", Online: &online, ObservedAt: now - 2000},
		{SampleID: "hot-new", DeviceID: "dev-hot-new", Source: "retention_test", Online: &online, ObservedAt: now - 500},
	}
	s.TelemetryWarm = []TelemetrySample{
		{SampleID: "warm-old", DeviceID: "dev-warm-old", Source: "retention_test", Online: &offline, ObservedAt: now - 4000},
		{SampleID: "warm-new", DeviceID: "dev-warm-new", Source: "retention_test", Online: &online, ObservedAt: now - 2000},
	}
	s.TelemetryCold = []TelemetrySample{
		{SampleID: "cold-expired", DeviceID: "dev-cold-expired", Source: "retention_test", Online: &offline, ObservedAt: now - 7000},
		{SampleID: "cold-keep", DeviceID: "dev-cold-keep", Source: "retention_test", Online: &online, ObservedAt: now - 5000},
	}
	s.applyTelemetryRetentionLocked(now)
	hot := append([]TelemetrySample(nil), s.TelemetryHot...)
	warm := append([]TelemetrySample(nil), s.TelemetryWarm...)
	cold := append([]TelemetrySample(nil), s.TelemetryCold...)
	s.mu.Unlock()

	if len(hot) != 1 || hot[0].SampleID != "hot-new" {
		t.Fatalf("expected only hot-new in hot tier, got=%v", hot)
	}

	warmIDs := telemetrySampleIDs(warm)
	if len(warmIDs) != 2 {
		t.Fatalf("expected 2 warm samples after promotion, got=%d", len(warmIDs))
	}
	if _, ok := warmIDs["warm-new"]; !ok {
		t.Fatalf("expected warm-new to remain in warm tier")
	}
	if _, ok := warmIDs["hot-old"]; !ok {
		t.Fatalf("expected hot-old to promote into warm tier")
	}

	coldIDs := telemetrySampleIDs(cold)
	if len(coldIDs) != 2 {
		t.Fatalf("expected 2 cold samples after prune/promotion, got=%d", len(coldIDs))
	}
	if _, ok := coldIDs["cold-keep"]; !ok {
		t.Fatalf("expected cold-keep in cold tier")
	}
	if _, ok := coldIDs["warm-old"]; !ok {
		t.Fatalf("expected warm-old promoted into cold tier")
	}
	if _, ok := coldIDs["cold-expired"]; ok {
		t.Fatalf("expected cold-expired sample pruned from cold tier")
	}
}

func TestTelemetryRetentionEnforcesTierCaps(t *testing.T) {
	s := LoadStore("")
	now := int64(10_000)
	online := true

	s.mu.Lock()
	s.TelemetryRetentionPolicy = TelemetryRetentionPolicy{
		HotRetentionMs:  100_000,
		WarmRetentionMs: 200_000,
		ColdRetentionMs: 300_000,
		HotMaxSamples:   2,
		WarmMaxSamples:  2,
		ColdMaxSamples:  2,
	}
	s.TelemetryHot = []TelemetrySample{
		{SampleID: "h1", DeviceID: "d1", Source: "retention_test", Online: &online, ObservedAt: now - 3},
		{SampleID: "h2", DeviceID: "d2", Source: "retention_test", Online: &online, ObservedAt: now - 2},
		{SampleID: "h3", DeviceID: "d3", Source: "retention_test", Online: &online, ObservedAt: now - 1},
	}
	s.TelemetryWarm = []TelemetrySample{
		{SampleID: "w1", DeviceID: "d1", Source: "retention_test", Online: &online, ObservedAt: now - 3},
		{SampleID: "w2", DeviceID: "d2", Source: "retention_test", Online: &online, ObservedAt: now - 2},
		{SampleID: "w3", DeviceID: "d3", Source: "retention_test", Online: &online, ObservedAt: now - 1},
	}
	s.TelemetryCold = []TelemetrySample{
		{SampleID: "c1", DeviceID: "d1", Source: "retention_test", Online: &online, ObservedAt: now - 3},
		{SampleID: "c2", DeviceID: "d2", Source: "retention_test", Online: &online, ObservedAt: now - 2},
		{SampleID: "c3", DeviceID: "d3", Source: "retention_test", Online: &online, ObservedAt: now - 1},
	}
	s.applyTelemetryRetentionLocked(now)
	hot := append([]TelemetrySample(nil), s.TelemetryHot...)
	warm := append([]TelemetrySample(nil), s.TelemetryWarm...)
	cold := append([]TelemetrySample(nil), s.TelemetryCold...)
	s.mu.Unlock()

	if len(hot) != 2 || hot[0].SampleID != "h2" || hot[1].SampleID != "h3" {
		t.Fatalf("expected hot tier capped to newest 2 samples [h2 h3], got=%v", hot)
	}
	if len(warm) != 2 || warm[0].SampleID != "w2" || warm[1].SampleID != "w3" {
		t.Fatalf("expected warm tier capped to newest 2 samples [w2 w3], got=%v", warm)
	}
	if len(cold) != 2 || cold[0].SampleID != "c2" || cold[1].SampleID != "c3" {
		t.Fatalf("expected cold tier capped to newest 2 samples [c2 c3], got=%v", cold)
	}
}

func TestTelemetrySamplingGovernorDropsFrequentSamplesByClass(t *testing.T) {
	s := LoadStore("")
	online := true
	offline := false

	s.mu.Lock()
	s.TelemetryGovernorRules = normalizeTelemetryGovernorRules([]TelemetryClassGovernorRule{
		{
			DeviceClass:         "access",
			MinSampleIntervalMs: int64((1 * time.Hour) / time.Millisecond),
			QueuePriority:       2,
			Roles:               []string{"ap"},
		},
		{
			DeviceClass:         "default",
			MinSampleIntervalMs: int64((1 * time.Hour) / time.Millisecond),
			QueuePriority:       9,
			Roles:               []string{"device"},
		},
	})
	s.TelemetryHot = nil
	s.TelemetryWarm = nil
	s.TelemetryCold = nil
	s.TelemetryLastByDevice = map[string]int64{}
	s.TelemetryAcceptedSamples = 0
	s.TelemetryDroppedSamples = 0
	s.mu.Unlock()

	req := TelemetryIngestRequest{
		Source:   "sampling_governor_test",
		DeviceID: "sample-ap-1",
		Device:   "Sample AP 1",
		Role:     "ap",
		Online:   &online,
	}
	_, _, firstDecision, ok := s.IngestTelemetryWithDecision(req)
	if !ok {
		t.Fatalf("first ingest failed")
	}
	if !firstDecision.Accepted {
		t.Fatalf("expected first ingest accepted, decision=%+v", firstDecision)
	}
	_, _, secondDecision, ok := s.IngestTelemetryWithDecision(req)
	if !ok {
		t.Fatalf("second ingest failed")
	}
	if secondDecision.Accepted {
		t.Fatalf("expected second ingest dropped by governor, decision=%+v", secondDecision)
	}

	req.EventType = "device_down"
	req.Online = &offline
	_, incident, thirdDecision, ok := s.IngestTelemetryWithDecision(req)
	if !ok {
		t.Fatalf("third ingest failed")
	}
	if !thirdDecision.Accepted {
		t.Fatalf("expected transition ingest accepted, decision=%+v", thirdDecision)
	}
	if incident == nil {
		t.Fatalf("expected offline transition to create incident")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.TelemetryHot) != 2 {
		t.Fatalf("expected 2 accepted telemetry samples in hot tier, got=%d", len(s.TelemetryHot))
	}
	if s.TelemetryAcceptedSamples != 2 {
		t.Fatalf("expected accepted_samples=2, got=%d", s.TelemetryAcceptedSamples)
	}
	if s.TelemetryDroppedSamples != 1 {
		t.Fatalf("expected dropped_samples=1, got=%d", s.TelemetryDroppedSamples)
	}
}

func TestPrioritizeTelemetryQueueByClassPriority(t *testing.T) {
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
	s.mu.Unlock()

	ordered := s.PrioritizeTelemetryQueue([]TelemetryIngestRequest{
		{DeviceID: "edge-1", Role: "station"},
		{DeviceID: "core-1", Role: "gateway"},
		{DeviceID: "access-1", Role: "switch"},
	})
	if len(ordered) != 3 {
		t.Fatalf("expected 3 prioritized events, got=%d", len(ordered))
	}
	if ordered[0].DeviceID != "core-1" || ordered[1].DeviceID != "access-1" || ordered[2].DeviceID != "edge-1" {
		t.Fatalf("unexpected queue order: %#v", ordered)
	}
}

func TestDetectTelemetryGapsCreatesAndResolvesIncidents(t *testing.T) {
	s := LoadStore("")
	online := true
	req := TelemetryIngestRequest{
		Source:   "gap_detector_test",
		DeviceID: "gap-node-1",
		Device:   "Gap Node 1",
		Role:     "gateway",
		Online:   &online,
	}
	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("initial ingest failed")
	}

	nowMs := time.Now().UnixMilli()
	s.mu.Lock()
	for i := range s.Devices {
		if s.Devices[i].ID == "gap-node-1" {
			s.Devices[i].LastSeen = nowMs - int64((45*time.Minute)/time.Millisecond)
		}
	}
	s.mu.Unlock()

	created, resolved := s.DetectTelemetryGaps(nowMs)
	if created != 1 || resolved != 0 {
		t.Fatalf("expected gap detector create=1 resolved=0, got create=%d resolved=%d", created, resolved)
	}

	s.mu.RLock()
	activeGap := false
	for _, inc := range s.Incidents {
		if inc.DeviceID == "gap-node-1" && inc.Type == "telemetry_gap" && inc.Resolved == nil {
			activeGap = true
			break
		}
	}
	s.mu.RUnlock()
	if !activeGap {
		t.Fatalf("expected active telemetry_gap incident")
	}

	s.mu.Lock()
	for i := range s.Devices {
		if s.Devices[i].ID == "gap-node-1" {
			s.Devices[i].LastSeen = nowMs
		}
	}
	s.mu.Unlock()

	created, resolved = s.DetectTelemetryGaps(nowMs)
	if created != 0 || resolved != 1 {
		t.Fatalf("expected gap detector create=0 resolved=1, got create=%d resolved=%d", created, resolved)
	}
}

func TestIngestTelemetryNormalizesSkewedObservedTimestamp(t *testing.T) {
	s := LoadStore("")
	online := true
	ingestStart := time.Now().UnixMilli()
	sourceObserved := ingestStart + int64((3*time.Hour)/time.Millisecond)
	req := TelemetryIngestRequest{
		Source:       "skew_test",
		EventType:    "telemetry",
		ObservedAtMs: sourceObserved,
		DeviceID:     "skew-device-1",
		Device:       "Skew Device 1",
		Role:         "gateway",
		SiteID:       "site-skew",
		Online:       &online,
	}

	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("ingest failed")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.TelemetryHot) == 0 {
		t.Fatalf("expected telemetry sample after ingest")
	}
	sample := s.TelemetryHot[len(s.TelemetryHot)-1]
	if !sample.TimestampCorrected {
		t.Fatalf("expected timestamp correction for large future skew")
	}
	if sample.SourceObservedAt != sourceObserved {
		t.Fatalf("expected source observed timestamp retained, got=%d want=%d", sample.SourceObservedAt, sourceObserved)
	}
	if sample.ClockSkewMs <= 0 {
		t.Fatalf("expected positive clock skew, got=%d", sample.ClockSkewMs)
	}
	if sample.TimestampConfidence > 0.2 {
		t.Fatalf("expected low confidence for corrected timestamp, got=%f", sample.TimestampConfidence)
	}
	if sample.ObservedAt < ingestStart-5000 || sample.ObservedAt > time.Now().UnixMilli()+5000 {
		t.Fatalf("expected normalized observed_at near ingest time, got=%d", sample.ObservedAt)
	}
}

func TestIngestTelemetryUsesObservedAtStringWhenValid(t *testing.T) {
	s := LoadStore("")
	online := true
	observed := time.Now().Add(-25 * time.Second).UTC()
	req := TelemetryIngestRequest{
		Source:     "skew_test_string",
		EventType:  "telemetry",
		ObservedAt: observed.Format(time.RFC3339),
		DeviceID:   "skew-device-2",
		Device:     "Skew Device 2",
		Role:       "switch",
		SiteID:     "site-skew",
		Online:     &online,
	}
	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("ingest failed")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.TelemetryHot) == 0 {
		t.Fatalf("expected telemetry sample")
	}
	sample := s.TelemetryHot[len(s.TelemetryHot)-1]
	if sample.TimestampCorrected {
		t.Fatalf("did not expect correction for small skew")
	}
	expectedMs := observed.UnixMilli()
	if absInt64(sample.ObservedAt-expectedMs) > 1500 {
		t.Fatalf("expected observed_at to use source timestamp, got=%d want~=%d", sample.ObservedAt, expectedMs)
	}
	if sample.TimestampConfidence < 0.85 {
		t.Fatalf("expected confidence >= 0.85 for small skew, got=%f", sample.TimestampConfidence)
	}
}

func TestTelemetryQualityReportAggregatesSourceStats(t *testing.T) {
	s := LoadStore("")
	online := true
	s.mu.Lock()
	s.TelemetryGovernorRules = normalizeTelemetryGovernorRules([]TelemetryClassGovernorRule{
		{
			DeviceClass:         "access",
			MinSampleIntervalMs: int64((1 * time.Hour) / time.Millisecond),
			QueuePriority:       1,
			Roles:               []string{"ap"},
		},
		{
			DeviceClass:         "default",
			MinSampleIntervalMs: int64((1 * time.Hour) / time.Millisecond),
			QueuePriority:       9,
			Roles:               []string{"device"},
		},
	})
	s.mu.Unlock()

	req := TelemetryIngestRequest{
		Source:   "quality_test",
		DeviceID: "quality-device-1",
		Device:   "Quality Device 1",
		Role:     "ap",
		SiteID:   "quality-site",
		Online:   &online,
	}
	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("first ingest failed")
	}
	if _, _, ok := s.IngestTelemetry(req); !ok {
		t.Fatalf("second ingest failed")
	}
	nowMs := time.Now().UnixMilli()
	s.RecordSourcePollOutcome("quality_test", true, "", nowMs)
	s.RecordSourcePollOutcome("quality_test", false, "timeout", nowMs)

	report := s.TelemetryQualityReport()
	if report.Health.SourceCount < 1 {
		t.Fatalf("expected at least one source in quality report")
	}
	card, ok := findQualityCard(report.Scorecards, "quality_test")
	if !ok {
		t.Fatalf("expected quality scorecard for source quality_test")
	}
	if card.Stats.TotalSamples != 2 {
		t.Fatalf("expected total samples=2, got=%d", card.Stats.TotalSamples)
	}
	if card.Stats.AcceptedSamples != 1 {
		t.Fatalf("expected accepted samples=1, got=%d", card.Stats.AcceptedSamples)
	}
	if card.Stats.DroppedSamples != 1 {
		t.Fatalf("expected dropped samples=1, got=%d", card.Stats.DroppedSamples)
	}
	if card.Stats.PollAttempts != 2 || card.Stats.PollFailures != 1 {
		t.Fatalf("expected poll attempts/failures 2/1, got %d/%d", card.Stats.PollAttempts, card.Stats.PollFailures)
	}
	if card.ErrorRatePct < 49.9 || card.ErrorRatePct > 50.1 {
		t.Fatalf("expected error_rate_pct around 50, got=%f", card.ErrorRatePct)
	}
}

func TestTelemetryBaselineReportBuildsRoleSiteMetrics(t *testing.T) {
	s := LoadStore("")
	online := true
	offline := false
	base := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)

	for i := 0; i < 12; i++ {
		latency := 20.0 + float64(i)
		state := &online
		if i%4 == 0 {
			state = &offline
		}
		req := TelemetryIngestRequest{
			Source:       "baseline_test",
			DeviceID:     "baseline-ap-1",
			Device:       "Baseline AP 1",
			Role:         "ap",
			SiteID:       "site-baseline",
			ObservedAtMs: base.Add(time.Duration(i) * 5 * time.Minute).UnixMilli(),
			Online:       state,
			LatencyMs:    &latency,
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("ingest failed at index=%d", i)
		}
	}

	report := s.TelemetryBaselineReport(72)
	if report.GroupCount == 0 {
		t.Fatalf("expected non-empty baseline groups")
	}
	group, ok := findBaselineGroup(report.Groups, "ap", "site-baseline")
	if !ok {
		t.Fatalf("expected baseline group role=ap site=site-baseline")
	}
	if group.SampleCount < 12 {
		t.Fatalf("expected at least 12 samples in group, got=%d", group.SampleCount)
	}

	latencyMetric, ok := findBaselineMetric(group.Metrics, "latency_ms")
	if !ok {
		t.Fatalf("expected latency baseline metric")
	}
	if latencyMetric.SampleCount < 12 {
		t.Fatalf("expected latency sample_count>=12, got=%d", latencyMetric.SampleCount)
	}
	if latencyMetric.UpperBound <= latencyMetric.LowerBound {
		t.Fatalf("expected latency upper bound > lower bound, got upper=%f lower=%f", latencyMetric.UpperBound, latencyMetric.LowerBound)
	}

	availabilityMetric, ok := findBaselineMetric(group.Metrics, "availability_pct")
	if !ok {
		t.Fatalf("expected availability baseline metric")
	}
	if availabilityMetric.Mean <= 0 || availabilityMetric.Mean >= 100 {
		t.Fatalf("expected availability mean between 0 and 100, got=%f", availabilityMetric.Mean)
	}
}

func TestTelemetryBaselineReportBuildsDayHourAnomalyWindows(t *testing.T) {
	s := LoadStore("")
	online := true
	base := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Hour)
	day := base.Weekday()
	dayHourA := base.Hour()
	dayHourB := (base.Hour() + 1) % 24

	for i := 0; i < 6; i++ {
		latency := 10.0 + float64(i)
		req := TelemetryIngestRequest{
			Source:       "anomaly_window_test",
			DeviceID:     "window-switch-1",
			Device:       "Window Switch 1",
			Role:         "switch",
			SiteID:       "site-window",
			ObservedAtMs: base.Add(time.Duration(i) * 7 * time.Minute).UnixMilli(),
			Online:       &online,
			LatencyMs:    &latency,
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("window A ingest failed index=%d", i)
		}
	}
	for i := 0; i < 6; i++ {
		latency := 50.0 + float64(i)
		req := TelemetryIngestRequest{
			Source:       "anomaly_window_test",
			DeviceID:     "window-switch-1",
			Device:       "Window Switch 1",
			Role:         "switch",
			SiteID:       "site-window",
			ObservedAtMs: base.Add(1*time.Hour + time.Duration(i)*7*time.Minute).UnixMilli(),
			Online:       &online,
			LatencyMs:    &latency,
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("window B ingest failed index=%d", i)
		}
	}

	report := s.TelemetryBaselineReport(72)
	group, ok := findBaselineGroup(report.Groups, "switch", "site-window")
	if !ok {
		t.Fatalf("expected switch/site-window baseline group")
	}
	if len(group.Windows) < 2 {
		t.Fatalf("expected at least 2 anomaly windows, got=%d", len(group.Windows))
	}

	foundA := false
	foundB := false
	for _, window := range group.Windows {
		if window.DayOfWeek != int(day) {
			continue
		}
		if window.HourOfDay == dayHourA && window.SampleCount >= 6 {
			foundA = true
		}
		if window.HourOfDay == dayHourB && window.SampleCount >= 6 {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected both anomaly windows present, foundA=%v foundB=%v", foundA, foundB)
	}
}

func TestTelemetryAlertIntelligenceBuildsConfidenceImpactAndStormSummary(t *testing.T) {
	s := LoadStore("")
	online := true
	offline := false

	for i := 1; i <= 5; i++ {
		deviceID := fmt.Sprintf("storm-node-%d", i)
		name := fmt.Sprintf("Storm Node %d", i)
		mac := fmt.Sprintf("aa:bb:cc:00:10:%02d", i)
		req := TelemetryIngestRequest{
			Source:   "storm_test",
			DeviceID: deviceID,
			Device:   name,
			Role:     "gateway",
			SiteID:   "storm-site",
			Mac:      mac,
			Serial:   "SER-" + deviceID,
			Online:   &online,
		}
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("seed ingest failed for %s", deviceID)
		}
	}

	coreNeighbors := []TelemetryNeighborFact{
		{NeighborIdentityHint: "storm-node-2", Protocol: "lldp"},
		{NeighborIdentityHint: "storm-node-3", Protocol: "lldp"},
		{NeighborIdentityHint: "storm-node-4", Protocol: "lldp"},
		{NeighborIdentityHint: "storm-node-5", Protocol: "lldp"},
	}
	if _, _, ok := s.IngestTelemetry(TelemetryIngestRequest{
		Source:    "storm_test",
		DeviceID:  "storm-node-1",
		Device:    "Storm Node 1",
		Role:      "gateway",
		SiteID:    "storm-site",
		EventType: "telemetry",
		Online:    &online,
		Neighbors: coreNeighbors,
	}); !ok {
		t.Fatalf("core neighbor ingest failed")
	}

	for i := 1; i <= 5; i++ {
		deviceID := fmt.Sprintf("storm-node-%d", i)
		if _, incident, ok := s.IngestTelemetry(TelemetryIngestRequest{
			Source:    "storm_test",
			DeviceID:  deviceID,
			Device:    fmt.Sprintf("Storm Node %d", i),
			Role:      "gateway",
			SiteID:    "storm-site",
			EventType: "offline",
			Online:    &offline,
		}); !ok || incident == nil {
			t.Fatalf("expected offline incident for %s", deviceID)
		}
	}

	report := s.TelemetryAlertIntelligence(50, 30, 3)
	if report.ActiveCount < 5 {
		t.Fatalf("expected at least 5 active alerts, got=%d", report.ActiveCount)
	}
	if report.RawAlertCount < 5 {
		t.Fatalf("expected raw alert count >= 5, got=%d", report.RawAlertCount)
	}
	if report.SummarizedAlertCount >= report.RawAlertCount {
		t.Fatalf("expected summarized alerts < raw alerts, raw=%d summarized=%d", report.RawAlertCount, report.SummarizedAlertCount)
	}
	if len(report.StormBursts) == 0 {
		t.Fatalf("expected at least one storm burst summary")
	}

	coreAlert, ok := findAlertByDeviceID(report.Alerts, "storm-node-1")
	if !ok {
		t.Fatalf("expected alert record for storm-node-1")
	}
	if coreAlert.ConfidenceScore < 0.60 {
		t.Fatalf("expected confidence >= 0.60 for sustained offline core node, got=%f", coreAlert.ConfidenceScore)
	}
	if coreAlert.Impact.ManagedReach < 5 {
		t.Fatalf("expected impact managed reach >= 5, got=%d", coreAlert.Impact.ManagedReach)
	}
	if coreAlert.Impact.Scope == "" || coreAlert.Impact.Scope == "unknown" {
		t.Fatalf("expected non-empty impact scope, got=%s", coreAlert.Impact.Scope)
	}
}

func TestTelemetryAlertIntelligenceLowerConfidenceForConflictingSample(t *testing.T) {
	s := LoadStore("")
	now := time.Now().UTC()
	nowMs := now.UnixMilli()
	online := true
	s.mu.Lock()
	s.Devices = append(s.Devices, Device{
		ID:       "confidence-conflict-1",
		Name:     "Confidence Conflict 1",
		Role:     "ap",
		SiteID:   "conflict-site",
		Online:   true,
		Source:   "conflict_test",
		LastSeen: nowMs,
	})
	s.Incidents = append(s.Incidents, Incident{
		ID:       "inc-confidence-conflict",
		DeviceID: "confidence-conflict-1",
		Type:     "offline",
		Severity: "critical",
		Started:  now.Add(-2 * time.Minute).Format(time.RFC3339),
		Source:   "conflict_test",
	})
	s.TelemetryHot = append(s.TelemetryHot, TelemetrySample{
		SampleID:            "ts-confidence-conflict",
		DeviceID:            "confidence-conflict-1",
		Source:              "conflict_test",
		EventType:           "telemetry",
		DeviceRole:          "ap",
		SiteID:              "conflict-site",
		Online:              &online,
		ObservedAt:          nowMs,
		ObservedISO:         now.Format(time.RFC3339),
		TimestampConfidence: 0.25,
	})
	s.mu.Unlock()

	report := s.TelemetryAlertIntelligence(20, 30, 4)
	alert, ok := findAlertByDeviceID(report.Alerts, "confidence-conflict-1")
	if !ok {
		t.Fatalf("expected alert record for confidence-conflict-1")
	}
	if alert.ConfidenceLevel != "low" {
		t.Fatalf("expected low confidence level for conflicting online sample, got=%s", alert.ConfidenceLevel)
	}
	if alert.ConfidenceScore >= 0.60 {
		t.Fatalf("expected confidence score below 0.60 for conflict case, got=%f", alert.ConfidenceScore)
	}
}

func TestLoadStoreMigratesLegacyAndReplayStable(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "store.json")

	legacy := map[string]any{
		"version": 1,
		"devices": []map[string]any{
			{"id": "legacy-gw-1", "name": "Legacy Gateway", "role": "gateway", "site_id": "site-a", "online": true},
			{"id": "legacy-ap-1", "name": "Legacy AP", "role": "ap", "site_id": "site-a", "online": false},
		},
		"incidents": []any{},
		"users": []map[string]any{
			{"username": "admin", "password": "admin"},
		},
	}
	body, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write legacy store: %v", err)
	}

	first := LoadStore(path)
	if first.Version != storeSchemaVersion {
		t.Fatalf("expected migrated schema version=%d, got=%d", storeSchemaVersion, first.Version)
	}
	firstIdent := first.ListDeviceIdentities()
	if len(firstIdent) < 2 {
		t.Fatalf("expected identities backfilled from legacy devices, got=%d", len(firstIdent))
	}
	first.mu.RLock()
	firstTelemetryTotal := len(first.TelemetryHot) + len(first.TelemetryWarm) + len(first.TelemetryCold)
	first.mu.RUnlock()
	if firstTelemetryTotal == 0 {
		t.Fatalf("expected telemetry retention backfill during schema migration")
	}

	second := LoadStore(path)
	if second.Version != storeSchemaVersion {
		t.Fatalf("expected replay schema version=%d, got=%d", storeSchemaVersion, second.Version)
	}
	secondIdent := second.ListDeviceIdentities()
	if len(secondIdent) != len(firstIdent) {
		t.Fatalf("expected identity count stable across replay, first=%d second=%d", len(firstIdent), len(secondIdent))
	}
	seen := map[string]struct{}{}
	for _, ident := range secondIdent {
		if ident.IdentityID == "" {
			t.Fatalf("identity id empty on replay")
		}
		if _, ok := seen[ident.IdentityID]; ok {
			t.Fatalf("duplicate identity id after replay: %s", ident.IdentityID)
		}
		seen[ident.IdentityID] = struct{}{}
	}
}

func TestLoadStoreRepairsDuplicateIdentityIDs(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "store-dup.json")

	now := int64(1_700_000_000_000)
	dupStore := map[string]any{
		"version": storeSchemaVersion,
		"devices": []map[string]any{
			{"id": "dup-a", "name": "Device A", "role": "gateway", "site_id": "site-a", "online": true, "last_seen": now},
			{"id": "dup-b", "name": "Device B", "role": "switch", "site_id": "site-a", "online": true, "last_seen": now},
		},
		"device_identities": []map[string]any{
			{
				"identity_id":       "ident-dup",
				"primary_device_id": "dup-a",
				"name":              "Device A",
				"role":              "gateway",
				"site_id":           "site-a",
				"last_seen":         now,
				"created_at":        "2025-01-01T00:00:00Z",
				"updated_at":        "2025-01-01T00:00:00Z",
			},
			{
				"identity_id":       "ident-dup",
				"primary_device_id": "dup-b",
				"name":              "Device B",
				"role":              "switch",
				"site_id":           "site-a",
				"last_seen":         now,
				"created_at":        "2025-01-01T00:00:00Z",
				"updated_at":        "2025-01-01T00:00:00Z",
			},
		},
		"users": []map[string]any{
			{"username": "admin", "password": "admin"},
		},
	}
	body, err := json.Marshal(dupStore)
	if err != nil {
		t.Fatalf("marshal dup store: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write dup store: %v", err)
	}

	s := LoadStore(path)
	idents := s.ListDeviceIdentities()
	if len(idents) < 2 {
		t.Fatalf("expected duplicate repair to backfill unique identities from devices, got=%d", len(idents))
	}
	seen := map[string]struct{}{}
	for _, ident := range idents {
		if ident.IdentityID == "" {
			t.Fatalf("empty identity id after repair")
		}
		if _, ok := seen[ident.IdentityID]; ok {
			t.Fatalf("duplicate identity id remained after repair: %s", ident.IdentityID)
		}
		seen[ident.IdentityID] = struct{}{}
	}
}

func TestTopologyGraphBuildsResolvedAndUnresolvedEdges(t *testing.T) {
	s := LoadStore("")

	reqB := TelemetryIngestRequest{
		Source:   "topo_test",
		DeviceID: "topo-b",
		Device:   "Topo B",
		Hostname: "topo-b.local",
		Mac:      "dd:00:00:00:00:02",
		Serial:   "SER-TOPO-B",
		SiteID:   "site-topo",
	}
	reqA := TelemetryIngestRequest{
		Source:   "topo_test",
		DeviceID: "topo-a",
		Device:   "Topo A",
		Hostname: "topo-a.local",
		Mac:      "dd:00:00:00:00:01",
		Serial:   "SER-TOPO-A",
		SiteID:   "site-topo",
		Neighbors: []TelemetryNeighborFact{
			{
				LocalInterface:     "eth0",
				NeighborDeviceName: "Topo B",
				NeighborInterface:  "eth9",
				Protocol:           "lldp",
			},
			{
				LocalInterface:       "eth1",
				NeighborIdentityHint: "ghost-core",
				NeighborInterface:    "xe-0/0/1",
				Protocol:             "lldp",
			},
		},
	}

	if _, _, ok := s.IngestTelemetry(reqB); !ok {
		t.Fatalf("ingest reqB failed")
	}
	if _, _, ok := s.IngestTelemetry(reqA); !ok {
		t.Fatalf("ingest reqA failed")
	}

	identA := findIdentityByPrimary(t, s, "topo-a")
	identB := findIdentityByPrimary(t, s, "topo-b")
	nodeA := topologyNodeIDForIdentity(identA.IdentityID)
	nodeB := topologyNodeIDForIdentity(identB.IdentityID)

	edges, _, _ := s.ListTopologyEdges(200, identA.IdentityID)
	if len(edges) < 2 {
		t.Fatalf("expected at least 2 topology edges for source identity, got=%d", len(edges))
	}
	foundResolved := false
	foundUnresolved := false
	for _, edge := range edges {
		if edge.SourceIdentityID != identA.IdentityID {
			t.Fatalf("unexpected edge source identity: %+v", edge)
		}
		if edge.FromNodeID == nodeA && edge.ToNodeID == nodeB && edge.Resolved {
			foundResolved = true
		}
		if edge.FromNodeID == nodeA && !edge.Resolved {
			foundUnresolved = true
		}
	}
	if !foundResolved {
		t.Fatalf("expected resolved edge from topo-a to topo-b")
	}
	if !foundUnresolved {
		t.Fatalf("expected unresolved edge from topo-a to unknown neighbor")
	}

	nodes, _, _ := s.ListTopologyNodes(500, "")
	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes including unresolved neighbor, got=%d", len(nodes))
	}
	foundManagedA := false
	foundManagedB := false
	foundExternal := false
	for _, node := range nodes {
		if node.NodeID == nodeA && node.Kind == "managed" {
			foundManagedA = true
		}
		if node.NodeID == nodeB && node.Kind == "managed" {
			foundManagedB = true
		}
		if node.Kind == "external" {
			foundExternal = true
		}
	}
	if !foundManagedA || !foundManagedB || !foundExternal {
		t.Fatalf("missing expected topology nodes: managedA=%v managedB=%v external=%v", foundManagedA, foundManagedB, foundExternal)
	}

	siteNodes, _, _ := s.ListTopologyNodes(500, "site-topo")
	for _, node := range siteNodes {
		if node.Kind != "managed" {
			t.Fatalf("site filtered nodes should include only managed nodes, got=%+v", node)
		}
		if node.SiteID != "site-topo" {
			t.Fatalf("site filtered node has unexpected site_id: %+v", node)
		}
	}
	if len(siteNodes) < 2 {
		t.Fatalf("expected at least two managed nodes for site filter, got=%d", len(siteNodes))
	}

	health := s.TopologyHealth()
	if health.NodeCount < 3 {
		t.Fatalf("unexpected topology health node count: %+v", health)
	}
	if health.EdgeCount < 2 {
		t.Fatalf("unexpected topology health edge count: %+v", health)
	}
	if health.UnknownNeighborEdges < 1 {
		t.Fatalf("expected unknown neighbor edges in health: %+v", health)
	}
	if health.ManagedNodeCount < 2 {
		t.Fatalf("expected managed node count >=2 in health: %+v", health)
	}
}

func TestTopologyPathTraceFindsShortestPath(t *testing.T) {
	s := LoadStore("")

	seed := []TelemetryIngestRequest{
		{
			Source:   "path_test",
			DeviceID: "path-a",
			Device:   "Path A",
			Mac:      "ff:00:00:00:00:01",
			Serial:   "SER-PATH-A",
			Neighbors: []TelemetryNeighborFact{
				{NeighborDeviceName: "Path B", LocalInterface: "eth0", Protocol: "lldp"},
			},
		},
		{
			Source:   "path_test",
			DeviceID: "path-b",
			Device:   "Path B",
			Mac:      "ff:00:00:00:00:02",
			Serial:   "SER-PATH-B",
			Neighbors: []TelemetryNeighborFact{
				{NeighborDeviceName: "Path C", LocalInterface: "eth1", Protocol: "lldp"},
			},
		},
		{
			Source:   "path_test",
			DeviceID: "path-c",
			Device:   "Path C",
			Mac:      "ff:00:00:00:00:03",
			Serial:   "SER-PATH-C",
		},
	}
	for _, req := range seed {
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("ingest failed for %s", req.DeviceID)
		}
	}

	identA := findIdentityByPrimary(t, s, "path-a")
	identC := findIdentityByPrimary(t, s, "path-c")
	nodes, edges, found, msg := s.TraceTopologyPath(identA.IdentityID, identC.IdentityID, "", "")
	if !found {
		t.Fatalf("expected topology path found, msg=%s", msg)
	}
	if len(nodes) < 3 {
		t.Fatalf("expected at least 3 nodes in path, got=%d", len(nodes))
	}
	if len(edges) < 2 {
		t.Fatalf("expected at least 2 edges in path, got=%d", len(edges))
	}
	if nodes[0].IdentityID != identA.IdentityID {
		t.Fatalf("unexpected source node in path: %+v", nodes[0])
	}
	if nodes[len(nodes)-1].IdentityID != identC.IdentityID {
		t.Fatalf("unexpected target node in path: %+v", nodes[len(nodes)-1])
	}

	_, _, foundMissing, _ := s.TraceTopologyPath("", "", "ident:missing-source", "ident:missing-target")
	if foundMissing {
		t.Fatalf("expected missing node path lookup to fail")
	}
}

func TestHAPairWatcherEmitsFailoverAndRecoveryEvents(t *testing.T) {
	s := LoadStore("")

	online := true
	seed := []TelemetryIngestRequest{
		{
			Source:   "ha_test",
			DeviceID: "ha-a",
			Device:   "HA A",
			Role:     "gateway",
			SiteID:   "ha-site-1",
			Mac:      "aa:bb:cc:00:00:10",
			Serial:   "SER-HA-A",
			Online:   &online,
		},
		{
			Source:   "ha_test",
			DeviceID: "ha-b",
			Device:   "HA B",
			Role:     "gateway",
			SiteID:   "ha-site-1",
			Mac:      "aa:bb:cc:00:00:20",
			Serial:   "SER-HA-B",
			Online:   &online,
		},
	}
	for _, req := range seed {
		if _, _, ok := s.IngestTelemetry(req); !ok {
			t.Fatalf("seed ingest failed for %s", req.DeviceID)
		}
	}

	pairs, _, _ := s.ListHAPairs(20, "")
	if len(pairs) == 0 {
		t.Fatalf("expected at least one HA pair")
	}
	pair := pairs[0]
	if pair.State != "redundant" {
		t.Fatalf("expected initial HA state redundant, got=%s", pair.State)
	}

	offline := false
	if _, _, ok := s.IngestTelemetry(TelemetryIngestRequest{
		Source:    "ha_test",
		DeviceID:  "ha-b",
		Device:    "HA B",
		Role:      "gateway",
		SiteID:    "ha-site-1",
		EventType: "offline",
		Online:    &offline,
	}); !ok {
		t.Fatalf("offline ingest failed")
	}

	pairsAfterFailover, _, _ := s.ListHAPairs(20, "")
	if len(pairsAfterFailover) == 0 {
		t.Fatalf("expected HA pair after failover transition")
	}
	if pairsAfterFailover[0].State != "failover" {
		t.Fatalf("expected failover state after offline event, got=%s", pairsAfterFailover[0].State)
	}
	if pairsAfterFailover[0].ActiveIdentityID == "" {
		t.Fatalf("expected active identity during failover")
	}

	events, _, _ := s.ListHAFailoverEvents(20, pair.PairID, "")
	if len(events) == 0 {
		t.Fatalf("expected failover events for pair=%s", pair.PairID)
	}
	if events[0].EventType != "failover" && events[0].EventType != "state_change" {
		t.Fatalf("unexpected failover event type: %+v", events[0])
	}
	if events[0].ToState != "failover" {
		t.Fatalf("expected event to_state=failover, got=%s", events[0].ToState)
	}

	if _, _, ok := s.IngestTelemetry(TelemetryIngestRequest{
		Source:    "ha_test",
		DeviceID:  "ha-b",
		Device:    "HA B",
		Role:      "gateway",
		SiteID:    "ha-site-1",
		EventType: "online",
		Online:    &online,
	}); !ok {
		t.Fatalf("recovery ingest failed")
	}

	pairsAfterRecovery, _, _ := s.ListHAPairs(20, "")
	if len(pairsAfterRecovery) == 0 {
		t.Fatalf("expected HA pair after recovery transition")
	}
	if pairsAfterRecovery[0].State != "redundant" {
		t.Fatalf("expected redundant state after recovery, got=%s", pairsAfterRecovery[0].State)
	}

	eventsAfterRecovery, _, _ := s.ListHAFailoverEvents(20, pair.PairID, "")
	if len(eventsAfterRecovery) < 2 {
		t.Fatalf("expected at least two HA transition events, got=%d", len(eventsAfterRecovery))
	}
	if eventsAfterRecovery[0].ToState != "redundant" {
		t.Fatalf("expected latest event to_state=redundant, got=%s", eventsAfterRecovery[0].ToState)
	}
}

func TestTopologyFixtureTriangleResolvedPath(t *testing.T) {
	s := LoadStore("")
	seed := loadTopologyFixture(t, "triangle_resolved.json")
	ingestTelemetryBatch(t, s, seed)

	identA := findIdentityByPrimary(t, s, "tri-a")
	identB := findIdentityByPrimary(t, s, "tri-b")
	identC := findIdentityByPrimary(t, s, "tri-c")

	nodes, edges, found, msg := s.TraceTopologyPath(identA.IdentityID, identC.IdentityID, "", "")
	if !found {
		t.Fatalf("expected topology path found tri-a->tri-c, msg=%s", msg)
	}
	if len(nodes) < 2 || len(edges) < 1 {
		t.Fatalf("expected non-empty path result, nodes=%d edges=%d", len(nodes), len(edges))
	}

	allEdges, _, _ := s.ListTopologyEdges(200, "")
	nodeA := topologyNodeIDForIdentity(identA.IdentityID)
	nodeB := topologyNodeIDForIdentity(identB.IdentityID)
	nodeC := topologyNodeIDForIdentity(identC.IdentityID)
	resolvedPairs := map[string]bool{}
	for _, edge := range allEdges {
		if edge.Resolved {
			resolvedPairs[fmt.Sprintf("%s>%s", edge.FromNodeID, edge.ToNodeID)] = true
		}
	}
	if !resolvedPairs[fmt.Sprintf("%s>%s", nodeA, nodeB)] {
		t.Fatalf("expected resolved edge tri-a -> tri-b")
	}
	if !resolvedPairs[fmt.Sprintf("%s>%s", nodeA, nodeC)] {
		t.Fatalf("expected resolved edge tri-a -> tri-c")
	}

	health := s.TopologyHealth()
	if health.ManagedNodeCount < 3 {
		t.Fatalf("expected at least three managed nodes for triangle fixture, got=%d", health.ManagedNodeCount)
	}
	if health.UnknownNeighborEdges != 0 {
		t.Fatalf("expected zero unresolved edges for triangle fixture, got=%d", health.UnknownNeighborEdges)
	}
}

func TestTopologyFixtureBranchIncludesUnknownNeighbors(t *testing.T) {
	s := LoadStore("")
	seed := loadTopologyFixture(t, "branch_unresolved.json")
	ingestTelemetryBatch(t, s, seed)

	nodes, _, _ := s.ListTopologyNodes(200, "")
	foundExternal := false
	for _, node := range nodes {
		if node.Kind == "external" {
			foundExternal = true
			break
		}
	}
	if !foundExternal {
		t.Fatalf("expected external topology nodes from unresolved neighbors")
	}

	health := s.TopologyHealth()
	if health.UnknownNeighborEdges < 2 {
		t.Fatalf("expected at least two unresolved neighbor edges, got=%d", health.UnknownNeighborEdges)
	}
	if health.ManagedNodeCount < 2 {
		t.Fatalf("expected at least two managed nodes in branch fixture, got=%d", health.ManagedNodeCount)
	}
}

func TestRetentionPolicyCompactsWarmAndColdObservations(t *testing.T) {
	s := LoadStore("")
	now := time.Now().UnixMilli()

	s.mu.Lock()
	s.SourceObservations = nil
	for i := 0; i < 12; i++ {
		s.SourceObservations = append(s.SourceObservations, SourceObservation{
			ObservationID: fmt.Sprintf("hot-%d", i),
			IdentityID:    "ident-a",
			Source:        "test",
			DeviceID:      "dev-a",
			ObservedAt:    now - int64(i)*int64(time.Hour/time.Millisecond),
		})
	}
	for i := 0; i < 12; i++ {
		s.SourceObservations = append(s.SourceObservations, SourceObservation{
			ObservationID: fmt.Sprintf("warm-%d", i),
			IdentityID:    "ident-a",
			Source:        "test",
			DeviceID:      "dev-a",
			ObservedAt:    now - int64(36+i)*int64(time.Hour/time.Millisecond),
		})
	}
	for i := 0; i < 30; i++ {
		s.SourceObservations = append(s.SourceObservations, SourceObservation{
			ObservationID: fmt.Sprintf("cold-%d", i),
			IdentityID:    "ident-a",
			Source:        "test",
			DeviceID:      "dev-a",
			ObservedAt:    now - int64(8*24+i)*int64(time.Hour/time.Millisecond),
		})
	}
	for i := 0; i < 8; i++ {
		s.SourceObservations = append(s.SourceObservations, SourceObservation{
			ObservationID: fmt.Sprintf("drop-%d", i),
			IdentityID:    "ident-a",
			Source:        "test",
			DeviceID:      "dev-a",
			ObservedAt:    now - int64(45*24+i)*int64(time.Hour/time.Millisecond),
		})
	}

	summary := s.applyRetentionPolicyLocked(now)
	s.mu.Unlock()

	if summary.BeforeCount != 62 {
		t.Fatalf("expected before_count=62 got=%d", summary.BeforeCount)
	}
	if summary.DroppedCount <= 0 {
		t.Fatalf("expected dropped observations")
	}
	if summary.AfterCount >= summary.BeforeCount {
		t.Fatalf("expected compacted result after=%d before=%d", summary.AfterCount, summary.BeforeCount)
	}
	if len(summary.Tiers) != 3 {
		t.Fatalf("expected 3 tiers got=%d", len(summary.Tiers))
	}
	if summary.Tiers[0].RetainedCount != 12 {
		t.Fatalf("expected hot tier to retain all points got=%d", summary.Tiers[0].RetainedCount)
	}
	if summary.Tiers[1].RetainedCount != 4 {
		t.Fatalf("expected warm tier downsample to 4 got=%d", summary.Tiers[1].RetainedCount)
	}
	if summary.Tiers[2].RetainedCount != 3 {
		t.Fatalf("expected cold tier downsample to 3 got=%d", summary.Tiers[2].RetainedCount)
	}
}
