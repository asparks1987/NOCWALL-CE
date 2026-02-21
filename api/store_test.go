package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
