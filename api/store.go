package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	storeSchemaVersion      = 2
	maxSourceObservations   = 10000
	maxDriftSnapshots       = 4000
	maxDeviceInterfaces     = 20000
	maxNeighborLinks        = 20000
	identityBackfillSource  = "store_migration"
	defaultDeviceSourceName = "ingest"
)

var (
	ErrInvalidPrimary  = errors.New("invalid_primary_id")
	ErrNoSecondary     = errors.New("no_secondary_ids")
	ErrPrimaryNotFound = errors.New("primary_identity_not_found")
)

type Store struct {
	mu sync.RWMutex

	Version            int                   `json:"version"`
	Devices            []Device              `json:"devices"`
	Incidents          []Incident            `json:"incidents"`
	Agents             []Agent               `json:"agents"`
	PushTokens         []PushRegisterRequest `json:"push_tokens"`
	Users              []User                `json:"users"`
	DeviceIdentities   []DeviceIdentity      `json:"device_identities"`
	DeviceInterfaces   []DeviceInterface     `json:"device_interfaces"`
	NeighborLinks      []NeighborLink        `json:"neighbor_links"`
	HardwareProfiles   []HardwareProfile     `json:"hardware_profiles"`
	SourceObservations []SourceObservation   `json:"source_observations"`
	DriftSnapshots     []DriftSnapshot       `json:"drift_snapshots"`

	filePath      string
	identityIndex map[string]string
}

type storePersist struct {
	Version            int                   `json:"version"`
	Devices            []Device              `json:"devices"`
	Incidents          []Incident            `json:"incidents"`
	Agents             []Agent               `json:"agents"`
	PushTokens         []PushRegisterRequest `json:"push_tokens"`
	Users              []User                `json:"users"`
	DeviceIdentities   []DeviceIdentity      `json:"device_identities"`
	DeviceInterfaces   []DeviceInterface     `json:"device_interfaces"`
	NeighborLinks      []NeighborLink        `json:"neighbor_links"`
	HardwareProfiles   []HardwareProfile     `json:"hardware_profiles"`
	SourceObservations []SourceObservation   `json:"source_observations"`
	DriftSnapshots     []DriftSnapshot       `json:"drift_snapshots"`
}

func LoadStore(path string) *Store {
	s := &Store{
		Version:       storeSchemaVersion,
		Devices:       seedDevices(),
		Incidents:     seedIncidents(),
		Users:         []User{{Username: "admin", Password: "admin"}},
		filePath:      path,
		identityIndex: map[string]string{},
	}
	if path == "" {
		s.ensureDefaultsAndMigrateLocked()
		return s
	}

	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, s)
	}

	s.ensureDefaultsAndMigrateLocked()
	s.save()
	return s
}

func (s *Store) save() {
	if s.filePath == "" {
		return
	}

	s.mu.RLock()
	payload := storePersist{
		Version:            s.Version,
		Devices:            append([]Device(nil), s.Devices...),
		Incidents:          append([]Incident(nil), s.Incidents...),
		Agents:             append([]Agent(nil), s.Agents...),
		PushTokens:         append([]PushRegisterRequest(nil), s.PushTokens...),
		Users:              append([]User(nil), s.Users...),
		DeviceIdentities:   append([]DeviceIdentity(nil), s.DeviceIdentities...),
		DeviceInterfaces:   append([]DeviceInterface(nil), s.DeviceInterfaces...),
		NeighborLinks:      append([]NeighborLink(nil), s.NeighborLinks...),
		HardwareProfiles:   append([]HardwareProfile(nil), s.HardwareProfiles...),
		SourceObservations: append([]SourceObservation(nil), s.SourceObservations...),
		DriftSnapshots:     append([]DriftSnapshot(nil), s.DriftSnapshots...),
	}
	s.mu.RUnlock()

	b, _ := json.MarshalIndent(payload, "", "  ")
	_ = os.WriteFile(s.filePath, b, 0o644)
}

func (s *Store) ensureDefaultsAndMigrateLocked() {
	if s.identityIndex == nil {
		s.identityIndex = map[string]string{}
	}
	if s.Version <= 0 {
		s.Version = 1
	}
	if len(s.Devices) == 0 {
		s.Devices = seedDevices()
	}
	if len(s.Users) == 0 {
		s.Users = []User{{Username: "admin", Password: "admin"}}
	}

	if len(s.DeviceIdentities) == 0 && len(s.Devices) > 0 {
		s.backfillInventoryFromDevicesLocked(identityBackfillSource)
	}
	if s.hasDuplicateIdentityIDsLocked() {
		s.DeviceIdentities = nil
		s.DeviceInterfaces = nil
		s.NeighborLinks = nil
		s.HardwareProfiles = nil
		s.SourceObservations = nil
		s.backfillInventoryFromDevicesLocked(identityBackfillSource)
	}
	if len(s.DriftSnapshots) == 0 && len(s.DeviceIdentities) > 0 {
		nowMs := time.Now().UnixMilli()
		for _, ident := range s.DeviceIdentities {
			observedAt := ident.LastSeen
			if observedAt <= 0 {
				observedAt = nowMs
			}
			s.recordDriftSnapshotLocked(ident, observedAt)
		}
	}
	s.rebuildIdentityIndexLocked()

	if len(s.SourceObservations) > maxSourceObservations {
		s.SourceObservations = append([]SourceObservation(nil), s.SourceObservations[len(s.SourceObservations)-maxSourceObservations:]...)
	}
	if len(s.DriftSnapshots) > maxDriftSnapshots {
		s.DriftSnapshots = append([]DriftSnapshot(nil), s.DriftSnapshots[len(s.DriftSnapshots)-maxDriftSnapshots:]...)
	}
	if s.Version < storeSchemaVersion {
		s.Version = storeSchemaVersion
	}
}

func (s *Store) hasDuplicateIdentityIDsLocked() bool {
	seen := map[string]struct{}{}
	for _, ident := range s.DeviceIdentities {
		id := strings.TrimSpace(ident.IdentityID)
		if id == "" {
			return true
		}
		if _, ok := seen[id]; ok {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

func (s *Store) ListDevices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Device, len(s.Devices))
	copy(out, s.Devices)
	return out
}

func (s *Store) ListIncidents() []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Incident, len(s.Incidents))
	copy(out, s.Incidents)
	return out
}

func (s *Store) ListAgents() []Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Agent, len(s.Agents))
	copy(out, s.Agents)
	return out
}

func (s *Store) ListDeviceIdentities() []DeviceIdentity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DeviceIdentity, len(s.DeviceIdentities))
	for i := range s.DeviceIdentities {
		out[i] = s.DeviceIdentities[i]
		out[i].SourceRefs = append([]string(nil), s.DeviceIdentities[i].SourceRefs...)
	}
	return out
}

func (s *Store) ListSourceObservations(limit int, identityID string) ([]SourceObservation, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	filterID := strings.TrimSpace(identityID)
	filtered := make([]SourceObservation, 0, len(s.SourceObservations))
	for i := len(s.SourceObservations) - 1; i >= 0; i-- {
		obs := s.SourceObservations[i]
		if filterID != "" && obs.IdentityID != filterID {
			continue
		}
		filtered = append(filtered, obs)
		if len(filtered) >= limit {
			break
		}
	}

	total := 0
	if filterID == "" {
		total = len(s.SourceObservations)
	} else {
		for _, obs := range s.SourceObservations {
			if obs.IdentityID == filterID {
				total++
			}
		}
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListDriftSnapshots(limit int, identityID string) ([]DriftSnapshot, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	filterID := strings.TrimSpace(identityID)
	filtered := make([]DriftSnapshot, 0, len(s.DriftSnapshots))
	for i := len(s.DriftSnapshots) - 1; i >= 0; i-- {
		snap := s.DriftSnapshots[i]
		if filterID != "" && snap.IdentityID != filterID {
			continue
		}
		filtered = append(filtered, snap)
		if len(filtered) >= limit {
			break
		}
	}

	total := 0
	if filterID == "" {
		total = len(s.DriftSnapshots)
	} else {
		for _, snap := range s.DriftSnapshots {
			if snap.IdentityID == filterID {
				total++
			}
		}
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListDeviceInterfaces(limit int, identityID string) ([]DeviceInterface, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	filterID := strings.TrimSpace(identityID)
	filtered := make([]DeviceInterface, 0, len(s.DeviceInterfaces))
	for i := len(s.DeviceInterfaces) - 1; i >= 0; i-- {
		item := s.DeviceInterfaces[i]
		if filterID != "" && item.IdentityID != filterID {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}

	total := 0
	if filterID == "" {
		total = len(s.DeviceInterfaces)
	} else {
		for _, item := range s.DeviceInterfaces {
			if item.IdentityID == filterID {
				total++
			}
		}
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListNeighborLinks(limit int, identityID string) ([]NeighborLink, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	filterID := strings.TrimSpace(identityID)
	filtered := make([]NeighborLink, 0, len(s.NeighborLinks))
	for i := len(s.NeighborLinks) - 1; i >= 0; i-- {
		item := s.NeighborLinks[i]
		if filterID != "" && item.IdentityID != filterID {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}

	total := 0
	if filterID == "" {
		total = len(s.NeighborLinks)
	} else {
		for _, item := range s.NeighborLinks {
			if item.IdentityID == filterID {
				total++
			}
		}
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListLifecycleScores(limit int, identityID string) ([]LifecycleScore, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	filterID := strings.TrimSpace(identityID)
	scores := make([]LifecycleScore, 0, len(s.DeviceIdentities))
	nowMs := time.Now().UnixMilli()
	for _, ident := range s.DeviceIdentities {
		if filterID != "" && ident.IdentityID != filterID {
			continue
		}
		score := 100
		reasons := make([]string, 0, 4)

		if ident.Vendor == "" {
			score -= 20
			reasons = append(reasons, "missing_vendor")
		}
		if ident.Model == "" {
			score -= 20
			reasons = append(reasons, "missing_model")
		}
		age := nowMs - ident.LastSeen
		if age > int64((24 * time.Hour).Milliseconds()) {
			score -= 25
			reasons = append(reasons, "stale_last_seen_24h")
		}
		if age > int64((7 * 24 * time.Hour).Milliseconds()) {
			score -= 20
			reasons = append(reasons, "stale_last_seen_7d")
		}

		if score < 0 {
			score = 0
		}
		level := "low"
		if score < 60 {
			level = "high"
		} else if score < 80 {
			level = "medium"
		}

		scores = append(scores, LifecycleScore{
			IdentityID: ident.IdentityID,
			Score:      score,
			Level:      level,
			Reasons:    reasons,
		})
	}

	truncated := false
	if len(scores) > limit {
		scores = scores[:limit]
		truncated = true
	}
	return scores, truncated, limit
}

func (s *Store) ListTopologyNodes(limit int, siteID string) ([]TopologyNode, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	siteFilter := normalizeKeyToken(siteID)
	nodes, _, _ := s.buildTopologyGraphLocked()
	filtered := make([]TopologyNode, 0, len(nodes))
	for _, node := range nodes {
		if siteFilter != "" {
			// Site filtering applies to managed identities only.
			if node.Kind != "managed" || normalizeKeyToken(node.SiteID) != siteFilter {
				continue
			}
		}
		filtered = append(filtered, node)
		if len(filtered) >= limit {
			break
		}
	}
	truncated := len(nodes) > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListTopologyEdges(limit int, identityID string) ([]TopologyEdge, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	filterID := strings.TrimSpace(identityID)
	_, edges, _ := s.buildTopologyGraphLocked()
	filtered := make([]TopologyEdge, 0, len(edges))
	for _, edge := range edges {
		if filterID != "" && edge.SourceIdentityID != filterID {
			continue
		}
		filtered = append(filtered, edge)
		if len(filtered) >= limit {
			break
		}
	}
	total := len(edges)
	if filterID != "" {
		total = 0
		for _, edge := range edges {
			if edge.SourceIdentityID == filterID {
				total++
			}
		}
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) TopologyHealth() TopologyHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, _, health := s.buildTopologyGraphLocked()
	return health
}

func (s *Store) TraceTopologyPath(sourceIdentityID, targetIdentityID, sourceNodeID, targetNodeID string) ([]TopologyNode, []TopologyEdge, bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes, edges, _ := s.buildTopologyGraphLocked()
	nodeByID := make(map[string]TopologyNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.NodeID] = node
	}

	sourceNodeID = strings.TrimSpace(sourceNodeID)
	targetNodeID = strings.TrimSpace(targetNodeID)
	sourceIdentityID = strings.TrimSpace(sourceIdentityID)
	targetIdentityID = strings.TrimSpace(targetIdentityID)

	if sourceNodeID == "" && sourceIdentityID != "" {
		sourceNodeID = topologyNodeIDForIdentity(sourceIdentityID)
	}
	if targetNodeID == "" && targetIdentityID != "" {
		targetNodeID = topologyNodeIDForIdentity(targetIdentityID)
	}
	if sourceNodeID == "" || targetNodeID == "" {
		return nil, nil, false, "source and target are required"
	}
	if _, ok := nodeByID[sourceNodeID]; !ok {
		return nil, nil, false, "source node not found"
	}
	if _, ok := nodeByID[targetNodeID]; !ok {
		return nil, nil, false, "target node not found"
	}

	if sourceNodeID == targetNodeID {
		return []TopologyNode{nodeByID[sourceNodeID]}, nil, true, "source equals target"
	}

	adj := make(map[string][]string, len(nodes))
	edgeByPair := make(map[string]TopologyEdge, len(edges)*2)
	for _, edge := range edges {
		if edge.FromNodeID == "" || edge.ToNodeID == "" {
			continue
		}
		adj[edge.FromNodeID] = append(adj[edge.FromNodeID], edge.ToNodeID)
		adj[edge.ToNodeID] = append(adj[edge.ToNodeID], edge.FromNodeID)
		edgeByPair[topologyEdgePairKey(edge.FromNodeID, edge.ToNodeID)] = edge
		edgeByPair[topologyEdgePairKey(edge.ToNodeID, edge.FromNodeID)] = edge
	}

	queue := []string{sourceNodeID}
	visited := map[string]bool{sourceNodeID: true}
	parent := map[string]string{}
	found := false

	for len(queue) > 0 && !found {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adj[current] {
			if visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = current
			if next == targetNodeID {
				found = true
				break
			}
			queue = append(queue, next)
		}
	}

	if !found {
		return nil, nil, false, "no path found"
	}

	nodeIDs := []string{targetNodeID}
	for cur := targetNodeID; cur != sourceNodeID; {
		p, ok := parent[cur]
		if !ok || p == "" {
			return nil, nil, false, "path reconstruction failed"
		}
		nodeIDs = append(nodeIDs, p)
		cur = p
	}
	// reverse
	for i, j := 0, len(nodeIDs)-1; i < j; i, j = i+1, j-1 {
		nodeIDs[i], nodeIDs[j] = nodeIDs[j], nodeIDs[i]
	}

	pathNodes := make([]TopologyNode, 0, len(nodeIDs))
	pathEdges := make([]TopologyEdge, 0, max(0, len(nodeIDs)-1))
	for i, nodeID := range nodeIDs {
		pathNodes = append(pathNodes, nodeByID[nodeID])
		if i == 0 {
			continue
		}
		prev := nodeIDs[i-1]
		if edge, ok := edgeByPair[topologyEdgePairKey(prev, nodeID)]; ok {
			pathEdges = append(pathEdges, edge)
		}
	}

	return pathNodes, pathEdges, true, ""
}

func (s *Store) buildTopologyGraphLocked() ([]TopologyNode, []TopologyEdge, TopologyHealth) {
	nowMs := time.Now().UnixMilli()
	nodesByID := make(map[string]TopologyNode, len(s.DeviceIdentities))
	tokenToNode := make(map[string]string, len(s.DeviceIdentities)*5)
	managedNodeIDs := make([]string, 0, len(s.DeviceIdentities))

	for _, ident := range s.DeviceIdentities {
		identityID := strings.TrimSpace(ident.IdentityID)
		if identityID == "" {
			continue
		}
		nodeID := topologyNodeIDForIdentity(identityID)
		node := TopologyNode{
			NodeID:          nodeID,
			IdentityID:      identityID,
			Label:           firstNonEmpty(strings.TrimSpace(ident.Name), strings.TrimSpace(ident.PrimaryDeviceID), identityID),
			Role:            strings.TrimSpace(ident.Role),
			SiteID:          strings.TrimSpace(ident.SiteID),
			LastSeen:        ident.LastSeen,
			Kind:            "managed",
			SourceRefsCount: len(ident.SourceRefs),
		}
		nodesByID[nodeID] = node
		managedNodeIDs = append(managedNodeIDs, nodeID)
		for _, token := range topologyIdentityTokens(ident) {
			if token == "" {
				continue
			}
			if _, exists := tokenToNode[token]; !exists {
				tokenToNode[token] = nodeID
			}
		}
	}

	edgesByKey := make(map[string]TopologyEdge, len(s.NeighborLinks))
	degree := make(map[string]int, len(nodesByID))
	unknownNeighborEdges := 0

	for _, link := range s.NeighborLinks {
		sourceIdentity := strings.TrimSpace(link.IdentityID)
		if sourceIdentity == "" {
			continue
		}
		fromNodeID := topologyNodeIDForIdentity(sourceIdentity)
		if _, ok := nodesByID[fromNodeID]; !ok {
			// Keep source nodes visible even if identity data is stale/missing.
			nodesByID[fromNodeID] = TopologyNode{
				NodeID:     fromNodeID,
				IdentityID: sourceIdentity,
				Label:      sourceIdentity,
				Kind:       "managed",
			}
			managedNodeIDs = append(managedNodeIDs, fromNodeID)
		}

		toNodeID, resolved := resolveNeighborNodeID(link, tokenToNode)
		if toNodeID == "" {
			token := normalizeKeyToken(firstNonEmpty(link.NeighborIdentityHint, link.NeighborDeviceName, link.NeighborInterfaceHint, link.ID))
			if token == "" {
				token = "unknown"
			}
			toNodeID = "unresolved:" + token
			resolved = false
		}

		if _, ok := nodesByID[toNodeID]; !ok {
			kind := "external"
			if strings.HasPrefix(toNodeID, "ident:") {
				kind = "managed"
			}
			nodesByID[toNodeID] = TopologyNode{
				NodeID: toNodeID,
				Label:  firstNonEmpty(strings.TrimSpace(link.NeighborDeviceName), strings.TrimSpace(link.NeighborIdentityHint), toNodeID),
				Kind:   kind,
			}
		}

		key := normalizeKeyToken(strings.Join([]string{
			fromNodeID,
			toNodeID,
			strings.TrimSpace(link.LocalInterface),
			strings.TrimSpace(link.NeighborInterfaceHint),
			strings.TrimSpace(link.Protocol),
		}, "|"))
		if key == "" {
			continue
		}
		if _, exists := edgesByKey[key]; exists {
			continue
		}

		edge := TopologyEdge{
			EdgeID:             "edge-" + key,
			FromNodeID:         fromNodeID,
			ToNodeID:           toNodeID,
			SourceIdentityID:   sourceIdentity,
			TargetIdentityHint: strings.TrimSpace(link.NeighborIdentityHint),
			LocalInterface:     strings.TrimSpace(link.LocalInterface),
			NeighborInterface:  strings.TrimSpace(link.NeighborInterfaceHint),
			Protocol:           strings.TrimSpace(link.Protocol),
			Source:             strings.TrimSpace(link.Source),
			UpdatedAt:          strings.TrimSpace(link.UpdatedAt),
			Resolved:           resolved,
		}
		edgesByKey[key] = edge
		degree[fromNodeID]++
		degree[toNodeID]++
		if !resolved {
			unknownNeighborEdges++
		}
	}

	nodes := make([]TopologyNode, 0, len(nodesByID))
	for _, node := range nodesByID {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].NodeID < nodes[j].NodeID
	})

	edges := make([]TopologyEdge, 0, len(edgesByKey))
	for _, edge := range edgesByKey {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromNodeID != edges[j].FromNodeID {
			return edges[i].FromNodeID < edges[j].FromNodeID
		}
		if edges[i].ToNodeID != edges[j].ToNodeID {
			return edges[i].ToNodeID < edges[j].ToNodeID
		}
		return edges[i].EdgeID < edges[j].EdgeID
	})

	isolatedManaged := 0
	staleManaged24h := 0
	managedSeen := map[string]struct{}{}
	for _, nodeID := range managedNodeIDs {
		if _, ok := managedSeen[nodeID]; ok {
			continue
		}
		managedSeen[nodeID] = struct{}{}
		node, ok := nodesByID[nodeID]
		if !ok {
			continue
		}
		if degree[nodeID] == 0 {
			isolatedManaged++
		}
		if node.LastSeen > 0 && nowMs-node.LastSeen > int64((24*time.Hour).Milliseconds()) {
			staleManaged24h++
		}
	}

	nodeIDs := make([]string, 0, len(nodesByID))
	for nodeID := range nodesByID {
		nodeIDs = append(nodeIDs, nodeID)
	}
	components := topologyConnectedComponents(nodeIDs, edges)

	health := TopologyHealth{
		NodeCount:            len(nodes),
		ManagedNodeCount:     len(managedSeen),
		EdgeCount:            len(edges),
		UnknownNeighborEdges: unknownNeighborEdges,
		IsolatedManagedNodes: isolatedManaged,
		StaleManagedNodes24h: staleManaged24h,
		ConnectedComponents:  components,
	}
	return nodes, edges, health
}

func (s *Store) MergeIdentities(primaryID string, secondaryIDs []string) (DeviceIdentity, []string, error) {
	primaryID = strings.TrimSpace(primaryID)
	if primaryID == "" {
		return DeviceIdentity{}, nil, ErrInvalidPrimary
	}

	cleanSecondary := make([]string, 0, len(secondaryIDs))
	seen := map[string]struct{}{}
	for _, id := range secondaryIDs {
		v := strings.TrimSpace(id)
		if v == "" || v == primaryID {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		cleanSecondary = append(cleanSecondary, v)
	}
	if len(cleanSecondary) == 0 {
		return DeviceIdentity{}, nil, ErrNoSecondary
	}

	s.mu.Lock()
	if s.findIdentityIndexLocked(primaryID) < 0 {
		s.mu.Unlock()
		return DeviceIdentity{}, nil, ErrPrimaryNotFound
	}
	merged := make([]string, 0, len(cleanSecondary))
	for _, sid := range cleanSecondary {
		if s.findIdentityIndexLocked(sid) < 0 {
			continue
		}
		primaryID = s.mergeIdentitiesLocked(primaryID, sid)
		merged = append(merged, sid)
	}
	idx := s.findIdentityIndexLocked(primaryID)
	if idx < 0 {
		s.mu.Unlock()
		return DeviceIdentity{}, merged, ErrPrimaryNotFound
	}
	identity := s.DeviceIdentities[idx]
	s.mu.Unlock()

	s.save()
	return identity, merged, nil
}

func (s *Store) InventorySchema() InventorySchemaResponse {
	return InventorySchemaResponse{
		Version: storeSchemaVersion,
		DeviceIdentity: []string{
			"identity_id", "primary_device_id", "name", "role", "site_id",
			"hostname", "mac_address", "serial_number", "vendor", "model",
			"source_refs", "last_seen", "created_at", "updated_at",
		},
		DeviceInterface: []string{
			"id", "identity_id", "name", "admin_up", "oper_up", "rx_bps", "tx_bps", "error_rate", "source", "updated_at",
		},
		NeighborLink: []string{
			"id", "identity_id", "local_interface", "neighbor_identity_hint", "neighbor_device_name", "neighbor_interface_hint", "protocol", "source", "updated_at",
		},
		HardwareProfile: []string{
			"identity_id", "vendor", "model", "firmware_version", "hardware_revision", "updated_at",
		},
		Observation: []string{
			"observation_id", "identity_id", "source", "device_id", "name", "role", "site_id", "hostname", "mac_address",
			"serial_number", "vendor", "model", "online", "latency_ms", "observed_at",
		},
		Notes: map[string]string{
			"stitching": "identity keys use mac, serial, hostname+site, and source+device_id hints",
			"drift":     "drift snapshots hash identity attributes to detect config metadata changes",
			"scope":     "phase-1 schema is foundational and intentionally minimal",
		},
	}
}

func (s *Store) AckIncident(id string, minutes int) (Incident, bool) {
	s.mu.Lock()
	var out Incident
	found := false
	for i := range s.Incidents {
		if s.Incidents[i].ID == id {
			until := time.Now().Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339)
			s.Incidents[i].AckUntil = &until
			out = s.Incidents[i]
			found = true
			break
		}
	}
	s.mu.Unlock()

	if found {
		s.save()
	}
	return out, found
}

func (s *Store) RegisterPush(req PushRegisterRequest) {
	s.mu.Lock()
	s.PushTokens = append(s.PushTokens, req)
	s.mu.Unlock()
	s.save()
}

func (s *Store) RegisterAgent(req AgentRegisterRequest) Agent {
	now := time.Now().UnixMilli()
	agentID := strings.TrimSpace(req.ID)
	if agentID == "" {
		agentID = "agent-" + randomID()
	}

	agentName := strings.TrimSpace(req.Name)
	if agentName == "" {
		agentName = agentID
	}

	incoming := Agent{
		ID:           agentID,
		Name:         agentName,
		SiteID:       strings.TrimSpace(req.SiteID),
		Version:      strings.TrimSpace(req.Version),
		Capabilities: append([]string(nil), req.Capabilities...),
		LastSeen:     now,
		Status:       "online",
	}

	s.mu.Lock()
	updated := false
	for i := range s.Agents {
		if s.Agents[i].ID == agentID {
			s.Agents[i] = incoming
			updated = true
			break
		}
	}
	if !updated {
		s.Agents = append(s.Agents, incoming)
	}
	s.mu.Unlock()

	s.save()
	return incoming
}

func (s *Store) IngestTelemetry(req TelemetryIngestRequest) (Device, *Incident, bool) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return Device{}, nil, false
	}

	now := time.Now()
	nowMs := now.UnixMilli()
	eventType := strings.ToLower(strings.TrimSpace(req.EventType))
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = defaultDeviceSourceName
	}

	deviceName := strings.TrimSpace(req.Device)
	if deviceName == "" {
		deviceName = deviceID
	}

	deviceRole := strings.TrimSpace(req.Role)
	if deviceRole == "" {
		deviceRole = "device"
	}

	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		siteID = "default"
	}

	online := true
	if req.Online != nil {
		online = *req.Online
	} else if eventType == "device_down" || eventType == "offline" {
		online = false
	}

	s.mu.Lock()

	idx := -1
	for i := range s.Devices {
		if s.Devices[i].ID == deviceID {
			idx = i
			break
		}
	}

	if idx == -1 {
		s.Devices = append(s.Devices, Device{
			ID:       deviceID,
			Name:     deviceName,
			Role:     deviceRole,
			SiteID:   siteID,
			Online:   online,
			Source:   source,
			LastSeen: nowMs,
		})
		idx = len(s.Devices) - 1
	}

	s.Devices[idx].Name = deviceName
	s.Devices[idx].Role = deviceRole
	s.Devices[idx].SiteID = siteID
	s.Devices[idx].Online = online
	s.Devices[idx].LatencyMs = req.LatencyMs
	s.Devices[idx].Source = source
	s.Devices[idx].LastSeen = nowMs

	onlineState := online
	s.upsertIdentityFromTelemetryLocked(req, source, deviceName, deviceRole, siteID, nowMs, &onlineState)

	var created *Incident
	if !online || eventType == "device_down" || eventType == "offline" {
		var active *Incident
		for i := range s.Incidents {
			if s.Incidents[i].DeviceID == deviceID && s.Incidents[i].Resolved == nil {
				active = &s.Incidents[i]
				break
			}
		}
		if active == nil {
			inc := Incident{
				ID:       "inc-" + randomID(),
				DeviceID: deviceID,
				Type:     "offline",
				Severity: "critical",
				Started:  now.UTC().Format(time.RFC3339),
				Message:  strings.TrimSpace(req.Message),
				Source:   source,
			}
			s.Incidents = append(s.Incidents, inc)
			created = &inc
		}
	}

	if online || eventType == "device_up" || eventType == "online" {
		resolvedAt := now.UTC().Format(time.RFC3339)
		for i := range s.Incidents {
			if s.Incidents[i].DeviceID == deviceID && s.Incidents[i].Resolved == nil {
				s.Incidents[i].Resolved = &resolvedAt
			}
		}
	}

	deviceCopy := s.Devices[idx]
	s.mu.Unlock()

	s.save()
	return deviceCopy, created, true
}

func (s *Store) ValidateUser(username, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.Users {
		if u.Username == username && u.Password == password {
			return true
		}
	}
	return false
}

func (s *Store) backfillInventoryFromDevicesLocked(source string) {
	nowMs := time.Now().UnixMilli()
	for _, dev := range s.Devices {
		online := dev.Online
		req := TelemetryIngestRequest{
			Source:    source,
			DeviceID:  dev.ID,
			Device:    dev.Name,
			Role:      dev.Role,
			SiteID:    dev.SiteID,
			Online:    &online,
			LatencyMs: dev.LatencyMs,
		}
		s.upsertIdentityFromTelemetryLocked(req, source, dev.Name, dev.Role, dev.SiteID, nowMs, &online)
	}
}

func (s *Store) upsertIdentityFromTelemetryLocked(req TelemetryIngestRequest, source, deviceName, deviceRole, siteID string, nowMs int64, online *bool) {
	obs := SourceObservation{
		ObservationID: "obs-" + randomID(),
		Source:        source,
		DeviceID:      strings.TrimSpace(req.DeviceID),
		Name:          strings.TrimSpace(deviceName),
		Role:          strings.TrimSpace(deviceRole),
		SiteID:        strings.TrimSpace(siteID),
		Hostname:      normalizeKeyToken(req.Hostname),
		MacAddress:    normalizeKeyToken(req.Mac),
		SerialNumber:  normalizeKeyToken(req.Serial),
		Vendor:        strings.TrimSpace(req.Vendor),
		Model:         strings.TrimSpace(req.Model),
		Online:        online,
		LatencyMs:     req.LatencyMs,
		ObservedAt:    nowMs,
	}
	if obs.Name == "" {
		obs.Name = obs.DeviceID
	}

	identityID := s.resolveIdentityIDLocked(obs)
	if identityID == "" {
		identityID = "ident-" + randomID()
		nowISO := time.Now().UTC().Format(time.RFC3339)
		s.DeviceIdentities = append(s.DeviceIdentities, DeviceIdentity{
			IdentityID:      identityID,
			PrimaryDeviceID: obs.DeviceID,
			Name:            obs.Name,
			Role:            obs.Role,
			SiteID:          obs.SiteID,
			Hostname:        obs.Hostname,
			MacAddress:      obs.MacAddress,
			SerialNumber:    obs.SerialNumber,
			Vendor:          obs.Vendor,
			Model:           obs.Model,
			SourceRefs:      []string{source},
			LastSeen:        nowMs,
			CreatedAt:       nowISO,
			UpdatedAt:       nowISO,
		})
	}

	idx := s.findIdentityIndexLocked(identityID)
	if idx < 0 {
		return
	}
	identity := &s.DeviceIdentities[idx]
	identity.LastSeen = nowMs
	identity.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if identity.PrimaryDeviceID == "" {
		identity.PrimaryDeviceID = obs.DeviceID
	}
	if obs.Name != "" && identity.Name != obs.Name {
		identity.Name = obs.Name
	}
	if obs.Role != "" && identity.Role != obs.Role {
		identity.Role = obs.Role
	}
	if obs.SiteID != "" && identity.SiteID != obs.SiteID {
		identity.SiteID = obs.SiteID
	}
	if obs.Hostname != "" && identity.Hostname != obs.Hostname {
		identity.Hostname = obs.Hostname
	}
	if obs.MacAddress != "" && identity.MacAddress != obs.MacAddress {
		identity.MacAddress = obs.MacAddress
	}
	if obs.SerialNumber != "" && identity.SerialNumber != obs.SerialNumber {
		identity.SerialNumber = obs.SerialNumber
	}
	if obs.Vendor != "" && identity.Vendor != obs.Vendor {
		identity.Vendor = obs.Vendor
	}
	if obs.Model != "" && identity.Model != obs.Model {
		identity.Model = obs.Model
	}
	identity.SourceRefs = appendUnique(identity.SourceRefs, source)
	obs.IdentityID = identity.IdentityID

	s.recordDriftSnapshotLocked(*identity, nowMs)
	s.upsertHardwareProfileLocked(identity.IdentityID, obs.Vendor, obs.Model)
	s.upsertInterfaceFactsLocked(identity.IdentityID, source, req.Interfaces)
	s.upsertNeighborFactsLocked(identity.IdentityID, source, req.Neighbors)
	s.SourceObservations = append(s.SourceObservations, obs)
	if len(s.SourceObservations) > maxSourceObservations {
		s.SourceObservations = append([]SourceObservation(nil), s.SourceObservations[len(s.SourceObservations)-maxSourceObservations:]...)
	}

	for _, key := range identityKeysFromObservation(obs) {
		s.identityIndex[key] = identity.IdentityID
	}
}

func (s *Store) resolveIdentityIDLocked(obs SourceObservation) string {
	if s.identityIndex == nil {
		s.identityIndex = map[string]string{}
	}
	var matches []string
	for _, key := range identityKeysFromObservation(obs) {
		if id, ok := s.identityIndex[key]; ok && id != "" {
			matches = appendUnique(matches, id)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	primary := matches[0]
	if len(matches) > 1 {
		for _, secondary := range matches[1:] {
			primary = s.mergeIdentitiesLocked(primary, secondary)
		}
	}
	return primary
}

func (s *Store) mergeIdentitiesLocked(primaryID, secondaryID string) string {
	if primaryID == "" {
		return secondaryID
	}
	if secondaryID == "" || secondaryID == primaryID {
		return primaryID
	}

	primaryIdx := s.findIdentityIndexLocked(primaryID)
	secondaryIdx := s.findIdentityIndexLocked(secondaryID)
	if primaryIdx < 0 {
		return secondaryID
	}
	if secondaryIdx < 0 {
		return primaryID
	}

	primary := &s.DeviceIdentities[primaryIdx]
	secondary := s.DeviceIdentities[secondaryIdx]
	if primary.PrimaryDeviceID == "" {
		primary.PrimaryDeviceID = secondary.PrimaryDeviceID
	}
	if primary.Name == "" {
		primary.Name = secondary.Name
	}
	if primary.Role == "" {
		primary.Role = secondary.Role
	}
	if primary.SiteID == "" {
		primary.SiteID = secondary.SiteID
	}
	if primary.Hostname == "" {
		primary.Hostname = secondary.Hostname
	}
	if primary.MacAddress == "" {
		primary.MacAddress = secondary.MacAddress
	}
	if primary.SerialNumber == "" {
		primary.SerialNumber = secondary.SerialNumber
	}
	if primary.Vendor == "" {
		primary.Vendor = secondary.Vendor
	}
	if primary.Model == "" {
		primary.Model = secondary.Model
	}
	if secondary.LastSeen > primary.LastSeen {
		primary.LastSeen = secondary.LastSeen
	}
	primary.SourceRefs = appendUnique(primary.SourceRefs, secondary.SourceRefs...)
	primary.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	for i := range s.SourceObservations {
		if s.SourceObservations[i].IdentityID == secondaryID {
			s.SourceObservations[i].IdentityID = primaryID
		}
	}
	for i := range s.HardwareProfiles {
		if s.HardwareProfiles[i].IdentityID == secondaryID {
			s.HardwareProfiles[i].IdentityID = primaryID
		}
	}
	for i := range s.DeviceInterfaces {
		if s.DeviceInterfaces[i].IdentityID == secondaryID {
			s.DeviceInterfaces[i].IdentityID = primaryID
		}
	}
	for i := range s.NeighborLinks {
		if s.NeighborLinks[i].IdentityID == secondaryID {
			s.NeighborLinks[i].IdentityID = primaryID
		}
	}

	s.DeviceIdentities = append(s.DeviceIdentities[:secondaryIdx], s.DeviceIdentities[secondaryIdx+1:]...)
	s.rebuildIdentityIndexLocked()
	return primaryID
}

func (s *Store) rebuildIdentityIndexLocked() {
	s.identityIndex = map[string]string{}
	for _, ident := range s.DeviceIdentities {
		obs := SourceObservation{
			IdentityID:   ident.IdentityID,
			Source:       "",
			DeviceID:     ident.PrimaryDeviceID,
			Name:         ident.Name,
			Role:         ident.Role,
			SiteID:       ident.SiteID,
			Hostname:     ident.Hostname,
			MacAddress:   ident.MacAddress,
			SerialNumber: ident.SerialNumber,
		}
		for _, key := range identityKeysFromObservation(obs) {
			s.identityIndex[key] = ident.IdentityID
		}
	}
}

func (s *Store) findIdentityIndexLocked(identityID string) int {
	for i := range s.DeviceIdentities {
		if s.DeviceIdentities[i].IdentityID == identityID {
			return i
		}
	}
	return -1
}

func (s *Store) upsertHardwareProfileLocked(identityID, vendor, model string) {
	vendor = strings.TrimSpace(vendor)
	model = strings.TrimSpace(model)
	if identityID == "" || (vendor == "" && model == "") {
		return
	}
	for i := range s.HardwareProfiles {
		if s.HardwareProfiles[i].IdentityID != identityID {
			continue
		}
		if s.HardwareProfiles[i].Vendor == "" && vendor != "" {
			s.HardwareProfiles[i].Vendor = vendor
		}
		if s.HardwareProfiles[i].Model == "" && model != "" {
			s.HardwareProfiles[i].Model = model
		}
		s.HardwareProfiles[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return
	}
	s.HardwareProfiles = append(s.HardwareProfiles, HardwareProfile{
		IdentityID: identityID,
		Vendor:     vendor,
		Model:      model,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Store) upsertInterfaceFactsLocked(identityID, source string, facts []TelemetryInterfaceFact) {
	identityID = strings.TrimSpace(identityID)
	source = strings.TrimSpace(source)
	if identityID == "" || source == "" || len(facts) == 0 {
		return
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	incoming := make([]DeviceInterface, 0, len(facts))
	seen := map[string]struct{}{}
	for _, fact := range facts {
		name := strings.TrimSpace(fact.Name)
		if name == "" {
			continue
		}
		id := "if-" + normalizeKeyToken(identityID+"|"+source+"|"+name)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		incoming = append(incoming, DeviceInterface{
			ID:         id,
			IdentityID: identityID,
			Name:       name,
			AdminUp:    fact.AdminUp,
			OperUp:     fact.OperUp,
			RxBps:      fact.RxBps,
			TxBps:      fact.TxBps,
			ErrorRate:  fact.ErrorRate,
			Source:     source,
			UpdatedAt:  nowISO,
		})
		if len(incoming) >= 512 {
			break
		}
	}
	if len(incoming) == 0 {
		return
	}

	next := make([]DeviceInterface, 0, len(s.DeviceInterfaces)+len(incoming))
	for _, row := range s.DeviceInterfaces {
		if row.IdentityID == identityID && row.Source == source {
			continue
		}
		next = append(next, row)
	}
	next = append(next, incoming...)
	if len(next) > maxDeviceInterfaces {
		next = append([]DeviceInterface(nil), next[len(next)-maxDeviceInterfaces:]...)
	}
	s.DeviceInterfaces = next
}

func (s *Store) upsertNeighborFactsLocked(identityID, source string, facts []TelemetryNeighborFact) {
	identityID = strings.TrimSpace(identityID)
	source = strings.TrimSpace(source)
	if identityID == "" || source == "" || len(facts) == 0 {
		return
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	incoming := make([]NeighborLink, 0, len(facts))
	seen := map[string]struct{}{}
	for _, fact := range facts {
		localIf := strings.TrimSpace(fact.LocalInterface)
		neighborName := strings.TrimSpace(fact.NeighborDeviceName)
		neighborIf := strings.TrimSpace(fact.NeighborInterface)
		neighborHint := strings.TrimSpace(fact.NeighborIdentityHint)
		protocol := strings.TrimSpace(fact.Protocol)
		if localIf == "" && neighborName == "" && neighborIf == "" && neighborHint == "" {
			continue
		}
		key := normalizeKeyToken(identityID + "|" + source + "|" + localIf + "|" + neighborHint + "|" + neighborName + "|" + neighborIf + "|" + protocol)
		id := "nbr-" + key
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		incoming = append(incoming, NeighborLink{
			ID:                    id,
			IdentityID:            identityID,
			LocalInterface:        localIf,
			NeighborIdentityHint:  neighborHint,
			NeighborDeviceName:    neighborName,
			NeighborInterfaceHint: neighborIf,
			Protocol:              protocol,
			Source:                source,
			UpdatedAt:             nowISO,
		})
		if len(incoming) >= 512 {
			break
		}
	}
	if len(incoming) == 0 {
		return
	}

	next := make([]NeighborLink, 0, len(s.NeighborLinks)+len(incoming))
	for _, row := range s.NeighborLinks {
		if row.IdentityID == identityID && row.Source == source {
			continue
		}
		next = append(next, row)
	}
	next = append(next, incoming...)
	if len(next) > maxNeighborLinks {
		next = append([]NeighborLink(nil), next[len(next)-maxNeighborLinks:]...)
	}
	s.NeighborLinks = next
}

func (s *Store) recordDriftSnapshotLocked(identity DeviceIdentity, observedAt int64) {
	fingerprint, attributes := buildDriftFingerprint(identity)
	lastFingerprint := ""
	for i := len(s.DriftSnapshots) - 1; i >= 0; i-- {
		if s.DriftSnapshots[i].IdentityID == identity.IdentityID {
			lastFingerprint = s.DriftSnapshots[i].Fingerprint
			break
		}
	}
	if lastFingerprint == fingerprint {
		return
	}

	snapshot := DriftSnapshot{
		SnapshotID:    "drift-" + randomID(),
		IdentityID:    identity.IdentityID,
		Fingerprint:   fingerprint,
		Changed:       lastFingerprint != "",
		ObservedAt:    observedAt,
		ObservedAtISO: time.UnixMilli(observedAt).UTC().Format(time.RFC3339),
		Attributes:    attributes,
	}
	s.DriftSnapshots = append(s.DriftSnapshots, snapshot)
	if len(s.DriftSnapshots) > maxDriftSnapshots {
		s.DriftSnapshots = append([]DriftSnapshot(nil), s.DriftSnapshots[len(s.DriftSnapshots)-maxDriftSnapshots:]...)
	}
}

func buildDriftFingerprint(identity DeviceIdentity) (string, map[string]string) {
	attrs := map[string]string{
		"primary_device_id": identity.PrimaryDeviceID,
		"name":              identity.Name,
		"role":              identity.Role,
		"site_id":           identity.SiteID,
		"hostname":          identity.Hostname,
		"mac_address":       identity.MacAddress,
		"serial_number":     identity.SerialNumber,
		"vendor":            identity.Vendor,
		"model":             identity.Model,
	}
	parts := []string{
		attrs["primary_device_id"],
		attrs["name"],
		attrs["role"],
		attrs["site_id"],
		attrs["hostname"],
		attrs["mac_address"],
		attrs["serial_number"],
		attrs["vendor"],
		attrs["model"],
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:]), attrs
}

func normalizeKeyToken(raw string) string {
	out := strings.ToLower(strings.TrimSpace(raw))
	out = strings.ReplaceAll(out, " ", "")
	return out
}

func appendUnique(values []string, incoming ...string) []string {
	if len(incoming) == 0 {
		return values
	}
	seen := map[string]struct{}{}
	for _, v := range values {
		if v == "" {
			continue
		}
		seen[v] = struct{}{}
	}
	for _, v := range incoming {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, v)
	}
	return values
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func topologyNodeIDForIdentity(identityID string) string {
	identityID = strings.TrimSpace(identityID)
	if identityID == "" {
		return ""
	}
	return "ident:" + identityID
}

func topologyIdentityTokens(ident DeviceIdentity) []string {
	tokens := []string{
		normalizeKeyToken(ident.IdentityID),
		normalizeKeyToken("identity:" + ident.IdentityID),
		normalizeKeyToken(ident.PrimaryDeviceID),
		normalizeKeyToken(ident.Name),
		normalizeKeyToken(ident.Hostname),
		normalizeKeyToken(ident.MacAddress),
		normalizeKeyToken(ident.SerialNumber),
	}
	out := make([]string, 0, len(tokens))
	seen := map[string]struct{}{}
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func resolveNeighborNodeID(link NeighborLink, tokenToNode map[string]string) (string, bool) {
	candidates := []string{
		normalizeKeyToken(link.NeighborIdentityHint),
		normalizeKeyToken("identity:" + link.NeighborIdentityHint),
		normalizeKeyToken(link.NeighborDeviceName),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if nodeID, ok := tokenToNode[candidate]; ok && nodeID != "" {
			return nodeID, true
		}
	}
	return "", false
}

func topologyConnectedComponents(nodeIDs []string, edges []TopologyEdge) int {
	if len(nodeIDs) == 0 {
		return 0
	}
	adj := make(map[string][]string, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		adj[nodeID] = nil
	}
	for _, edge := range edges {
		if edge.FromNodeID == "" || edge.ToNodeID == "" {
			continue
		}
		adj[edge.FromNodeID] = append(adj[edge.FromNodeID], edge.ToNodeID)
		adj[edge.ToNodeID] = append(adj[edge.ToNodeID], edge.FromNodeID)
	}

	visited := make(map[string]bool, len(nodeIDs))
	components := 0
	queue := make([]string, 0, len(nodeIDs))
	for _, start := range nodeIDs {
		if start == "" || visited[start] {
			continue
		}
		components++
		queue = queue[:0]
		queue = append(queue, start)
		visited[start] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, next := range adj[current] {
				if next == "" || visited[next] {
					continue
				}
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return components
}

func topologyEdgePairKey(fromNodeID, toNodeID string) string {
	return strings.TrimSpace(fromNodeID) + "->" + strings.TrimSpace(toNodeID)
}

func identityKeysFromObservation(obs SourceObservation) []string {
	keys := make([]string, 0, 8)
	deviceID := normalizeKeyToken(obs.DeviceID)
	source := normalizeKeyToken(obs.Source)
	mac := normalizeKeyToken(obs.MacAddress)
	serial := normalizeKeyToken(obs.SerialNumber)
	hostname := normalizeKeyToken(obs.Hostname)
	site := normalizeKeyToken(obs.SiteID)
	name := normalizeKeyToken(obs.Name)

	if source != "" && deviceID != "" {
		keys = append(keys, "source_device:"+source+"|"+deviceID)
	}
	if deviceID != "" {
		keys = append(keys, "device:"+deviceID)
	}
	if mac != "" {
		keys = append(keys, "mac:"+mac)
	}
	if serial != "" {
		keys = append(keys, "serial:"+serial)
	}
	if hostname != "" && site != "" {
		keys = append(keys, "host_site:"+site+"|"+hostname)
	}
	if name != "" && site != "" {
		keys = append(keys, "name_site:"+site+"|"+name)
	}
	return keys
}
