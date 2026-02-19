package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	storeSchemaVersion      = 2
	maxSourceObservations   = 10000
	maxDriftSnapshots       = 4000
	identityBackfillSource  = "store_migration"
	defaultDeviceSourceName = "ingest"
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
