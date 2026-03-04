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
	storeSchemaVersion      = 4
	maxSourceObservations   = 10000
	maxDriftSnapshots       = 4000
	maxDeviceInterfaces     = 20000
	maxNeighborLinks        = 20000
	maxHAFailoverEvents     = 4000
	maxSamplingStateDevices = 20000
	defaultHotRetentionMs   = int64((6 * time.Hour) / time.Millisecond)
	defaultWarmRetentionMs  = int64((7 * 24 * time.Hour) / time.Millisecond)
	defaultColdRetentionMs  = int64((90 * 24 * time.Hour) / time.Millisecond)
	defaultHotMaxSamples    = 5000
	defaultWarmMaxSamples   = 20000
	defaultColdMaxSamples   = 100000
	defaultCoreSampleMs     = int64((5 * time.Second) / time.Millisecond)
	defaultDistSampleMs     = int64((10 * time.Second) / time.Millisecond)
	defaultAccessSampleMs   = int64((15 * time.Second) / time.Millisecond)
	defaultEdgeSampleMs     = int64((30 * time.Second) / time.Millisecond)
	defaultGenericSampleMs  = int64((20 * time.Second) / time.Millisecond)
	telemetryGapMultiplier  = int64(4)
	minTelemetryGapMs       = int64((2 * time.Minute) / time.Millisecond)
	identityBackfillSource  = "store_migration"
	defaultDeviceSourceName = "ingest"
	retentionHotHours       = 24
	retentionWarmHours      = 7 * 24
	retentionColdHours      = 30 * 24
)

var (
	ErrInvalidPrimary  = errors.New("invalid_primary_id")
	ErrNoSecondary     = errors.New("no_secondary_ids")
	ErrPrimaryNotFound = errors.New("primary_identity_not_found")
)

type TelemetryRetentionPolicy struct {
	HotRetentionMs  int64 `json:"hot_retention_ms"`
	WarmRetentionMs int64 `json:"warm_retention_ms"`
	ColdRetentionMs int64 `json:"cold_retention_ms"`
	HotMaxSamples   int   `json:"hot_max_samples"`
	WarmMaxSamples  int   `json:"warm_max_samples"`
	ColdMaxSamples  int   `json:"cold_max_samples"`
}

type TelemetrySample struct {
	SampleID    string   `json:"sample_id"`
	DeviceID    string   `json:"device_id"`
	IdentityID  string   `json:"identity_id,omitempty"`
	Source      string   `json:"source"`
	EventType   string   `json:"event_type"`
	DeviceRole  string   `json:"device_role,omitempty"`
	SiteID      string   `json:"site_id,omitempty"`
	Online      *bool    `json:"online,omitempty"`
	LatencyMs   *float64 `json:"latency_ms,omitempty"`
	ObservedAt  int64    `json:"observed_at"`
	ObservedISO string   `json:"observed_at_iso,omitempty"`
}

type Store struct {
	mu sync.RWMutex

	Version                     int                          `json:"version"`
	Devices                     []Device                     `json:"devices"`
	Incidents                   []Incident                   `json:"incidents"`
	Agents                      []Agent                      `json:"agents"`
	PushTokens                  []PushRegisterRequest        `json:"push_tokens"`
	Users                       []User                       `json:"users"`
	DeviceIdentities            []DeviceIdentity             `json:"device_identities"`
	DeviceInterfaces            []DeviceInterface            `json:"device_interfaces"`
	NeighborLinks               []NeighborLink               `json:"neighbor_links"`
	HardwareProfiles            []HardwareProfile            `json:"hardware_profiles"`
	SourceObservations          []SourceObservation          `json:"source_observations"`
	DriftSnapshots              []DriftSnapshot              `json:"drift_snapshots"`
	HAPairs                     []HAPairStatus               `json:"ha_pairs"`
	HAFailoverEvents            []HAFailoverEvent            `json:"ha_failover_events"`
	TelemetryRetentionPolicy    TelemetryRetentionPolicy     `json:"telemetry_retention_policy"`
	TelemetryGovernorRules      []TelemetryClassGovernorRule `json:"telemetry_governor_rules"`
	TelemetryAcceptedSamples    int64                        `json:"telemetry_accepted_samples"`
	TelemetryDroppedSamples     int64                        `json:"telemetry_dropped_samples"`
	TelemetryGovernorLastEvalMs int64                        `json:"telemetry_governor_last_eval_ms"`
	TelemetryHot                []TelemetrySample            `json:"telemetry_hot"`
	TelemetryWarm               []TelemetrySample            `json:"telemetry_warm"`
	TelemetryCold               []TelemetrySample            `json:"telemetry_cold"`
	TelemetryLastByDevice       map[string]int64             `json:"telemetry_last_by_device,omitempty"`

	filePath      string
	identityIndex map[string]string
	retentionLast TelemetryRetentionSummary
}

type storePersist struct {
	Version                     int                          `json:"version"`
	Devices                     []Device                     `json:"devices"`
	Incidents                   []Incident                   `json:"incidents"`
	Agents                      []Agent                      `json:"agents"`
	PushTokens                  []PushRegisterRequest        `json:"push_tokens"`
	Users                       []User                       `json:"users"`
	DeviceIdentities            []DeviceIdentity             `json:"device_identities"`
	DeviceInterfaces            []DeviceInterface            `json:"device_interfaces"`
	NeighborLinks               []NeighborLink               `json:"neighbor_links"`
	HardwareProfiles            []HardwareProfile            `json:"hardware_profiles"`
	SourceObservations          []SourceObservation          `json:"source_observations"`
	DriftSnapshots              []DriftSnapshot              `json:"drift_snapshots"`
	HAPairs                     []HAPairStatus               `json:"ha_pairs"`
	HAFailoverEvents            []HAFailoverEvent            `json:"ha_failover_events"`
	TelemetryRetentionPolicy    TelemetryRetentionPolicy     `json:"telemetry_retention_policy"`
	TelemetryGovernorRules      []TelemetryClassGovernorRule `json:"telemetry_governor_rules"`
	TelemetryAcceptedSamples    int64                        `json:"telemetry_accepted_samples"`
	TelemetryDroppedSamples     int64                        `json:"telemetry_dropped_samples"`
	TelemetryGovernorLastEvalMs int64                        `json:"telemetry_governor_last_eval_ms"`
	TelemetryHot                []TelemetrySample            `json:"telemetry_hot"`
	TelemetryWarm               []TelemetrySample            `json:"telemetry_warm"`
	TelemetryCold               []TelemetrySample            `json:"telemetry_cold"`
	TelemetryLastByDevice       map[string]int64             `json:"telemetry_last_by_device,omitempty"`
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
		Version:                     s.Version,
		Devices:                     append([]Device(nil), s.Devices...),
		Incidents:                   append([]Incident(nil), s.Incidents...),
		Agents:                      append([]Agent(nil), s.Agents...),
		PushTokens:                  append([]PushRegisterRequest(nil), s.PushTokens...),
		Users:                       append([]User(nil), s.Users...),
		DeviceIdentities:            append([]DeviceIdentity(nil), s.DeviceIdentities...),
		DeviceInterfaces:            append([]DeviceInterface(nil), s.DeviceInterfaces...),
		NeighborLinks:               append([]NeighborLink(nil), s.NeighborLinks...),
		HardwareProfiles:            append([]HardwareProfile(nil), s.HardwareProfiles...),
		SourceObservations:          append([]SourceObservation(nil), s.SourceObservations...),
		DriftSnapshots:              append([]DriftSnapshot(nil), s.DriftSnapshots...),
		HAPairs:                     append([]HAPairStatus(nil), s.HAPairs...),
		HAFailoverEvents:            append([]HAFailoverEvent(nil), s.HAFailoverEvents...),
		TelemetryRetentionPolicy:    s.TelemetryRetentionPolicy,
		TelemetryGovernorRules:      append([]TelemetryClassGovernorRule(nil), s.TelemetryGovernorRules...),
		TelemetryAcceptedSamples:    s.TelemetryAcceptedSamples,
		TelemetryDroppedSamples:     s.TelemetryDroppedSamples,
		TelemetryGovernorLastEvalMs: s.TelemetryGovernorLastEvalMs,
		TelemetryHot:                append([]TelemetrySample(nil), s.TelemetryHot...),
		TelemetryWarm:               append([]TelemetrySample(nil), s.TelemetryWarm...),
		TelemetryCold:               append([]TelemetrySample(nil), s.TelemetryCold...),
		TelemetryLastByDevice:       cloneStringInt64Map(s.TelemetryLastByDevice),
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
	s.TelemetryRetentionPolicy = normalizeTelemetryRetentionPolicy(s.TelemetryRetentionPolicy)
	s.TelemetryGovernorRules = normalizeTelemetryGovernorRules(s.TelemetryGovernorRules)
	if s.TelemetryLastByDevice == nil {
		s.TelemetryLastByDevice = map[string]int64{}
	}
	if s.TelemetryGovernorLastEvalMs <= 0 {
		s.TelemetryGovernorLastEvalMs = time.Now().UnixMilli()
	}
	if s.Version < storeSchemaVersion && len(s.TelemetryHot) == 0 && len(s.TelemetryWarm) == 0 && len(s.TelemetryCold) == 0 {
		s.backfillTelemetryRetentionFromObservationsLocked()
	}
	s.applyTelemetryRetentionLocked(time.Now().UnixMilli())
	s.pruneSamplingStateLocked(time.Now().UnixMilli())

	if len(s.SourceObservations) > maxSourceObservations {
		s.SourceObservations = append([]SourceObservation(nil), s.SourceObservations[len(s.SourceObservations)-maxSourceObservations:]...)
	}
	s.retentionLast = s.applyRetentionPolicyLocked(time.Now().UnixMilli())
	if len(s.DriftSnapshots) > maxDriftSnapshots {
		s.DriftSnapshots = append([]DriftSnapshot(nil), s.DriftSnapshots[len(s.DriftSnapshots)-maxDriftSnapshots:]...)
	}
	if len(s.HAFailoverEvents) > maxHAFailoverEvents {
		s.HAFailoverEvents = append([]HAFailoverEvent(nil), s.HAFailoverEvents[len(s.HAFailoverEvents)-maxHAFailoverEvents:]...)
	}
	s.updateHAPairWatcherLocked(time.Now().UnixMilli())
	if s.Version < storeSchemaVersion {
		s.Version = storeSchemaVersion
	}
}

func (s *Store) LastRetentionSummary() TelemetryRetentionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.retentionLast
	out.Tiers = append([]TelemetryRetentionTier(nil), s.retentionLast.Tiers...)
	return out
}

func (s *Store) TelemetryGovernorStatus() TelemetryGovernorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := TelemetryGovernorStatus{
		LastEvaluatedAtMs: s.TelemetryGovernorLastEvalMs,
		AcceptedSamples:   s.TelemetryAcceptedSamples,
		DroppedSamples:    s.TelemetryDroppedSamples,
		Rules:             append([]TelemetryClassGovernorRule(nil), s.TelemetryGovernorRules...),
	}
	activeGaps := 0
	for _, inc := range s.Incidents {
		if inc.Type == "telemetry_gap" && inc.Resolved == nil {
			activeGaps++
		}
	}
	status.ActiveGapIncidents = activeGaps
	return status
}

func (s *Store) PrioritizeTelemetryQueue(events []TelemetryIngestRequest) []TelemetryIngestRequest {
	if len(events) <= 1 {
		return append([]TelemetryIngestRequest(nil), events...)
	}
	s.mu.RLock()
	rules := append([]TelemetryClassGovernorRule(nil), s.TelemetryGovernorRules...)
	s.mu.RUnlock()

	type prioritizedEvent struct {
		req      TelemetryIngestRequest
		priority int
	}
	ordered := make([]prioritizedEvent, 0, len(events))
	for _, event := range events {
		rule := telemetryRuleForRole(event.Role, rules)
		ordered = append(ordered, prioritizedEvent{
			req:      event,
			priority: rule.QueuePriority,
		})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].priority < ordered[j].priority
	})

	out := make([]TelemetryIngestRequest, 0, len(ordered))
	for _, item := range ordered {
		out = append(out, item.req)
	}
	return out
}

func (s *Store) DetectTelemetryGaps(nowMs int64) (int, int) {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	s.mu.Lock()
	created, resolved, changed := s.applyTelemetryGapDetectionLocked(nowMs)
	s.mu.Unlock()
	if changed {
		s.save()
	}
	return created, resolved
}

func (s *Store) applyRetentionPolicyLocked(nowMs int64) TelemetryRetentionSummary {
	summary := TelemetryRetentionSummary{
		ObservedAtMs: nowMs,
		BeforeCount:  len(s.SourceObservations),
		Tiers: []TelemetryRetentionTier{
			{Name: "hot", MaxAgeHours: retentionHotHours, KeepEvery: 1},
			{Name: "warm", MaxAgeHours: retentionWarmHours, KeepEvery: 3},
			{Name: "cold", MaxAgeHours: retentionColdHours, KeepEvery: 10},
		},
	}
	if len(s.SourceObservations) == 0 {
		return summary
	}

	hotCutoff := nowMs - int64(retentionHotHours)*int64(time.Hour/time.Millisecond)
	warmCutoff := nowMs - int64(retentionWarmHours)*int64(time.Hour/time.Millisecond)
	coldCutoff := nowMs - int64(retentionColdHours)*int64(time.Hour/time.Millisecond)

	retained := make([]SourceObservation, 0, len(s.SourceObservations))
	warmIdx := 0
	coldIdx := 0
	for _, obs := range s.SourceObservations {
		if obs.ObservedAt >= hotCutoff {
			retained = append(retained, obs)
			summary.Tiers[0].RetainedCount++
			continue
		}
		if obs.ObservedAt >= warmCutoff {
			if warmIdx%summary.Tiers[1].KeepEvery == 0 {
				retained = append(retained, obs)
				summary.Tiers[1].RetainedCount++
			}
			warmIdx++
			continue
		}
		if obs.ObservedAt >= coldCutoff {
			if coldIdx%summary.Tiers[2].KeepEvery == 0 {
				retained = append(retained, obs)
				summary.Tiers[2].RetainedCount++
			}
			coldIdx++
		}
	}

	if len(retained) > maxSourceObservations {
		retained = append([]SourceObservation(nil), retained[len(retained)-maxSourceObservations:]...)
	}
	s.SourceObservations = retained
	summary.AfterCount = len(s.SourceObservations)
	summary.DroppedCount = summary.BeforeCount - summary.AfterCount
	return summary
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

func cloneStringInt64Map(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func defaultTelemetryGovernorRules() []TelemetryClassGovernorRule {
	return []TelemetryClassGovernorRule{
		{
			DeviceClass:         "core",
			MinSampleIntervalMs: defaultCoreSampleMs,
			QueuePriority:       0,
			Roles:               []string{"gateway", "router", "firewall", "controller", "core"},
		},
		{
			DeviceClass:         "distribution",
			MinSampleIntervalMs: defaultDistSampleMs,
			QueuePriority:       1,
			Roles:               []string{"distribution", "aggregation"},
		},
		{
			DeviceClass:         "access",
			MinSampleIntervalMs: defaultAccessSampleMs,
			QueuePriority:       2,
			Roles:               []string{"switch", "ap", "access", "bridge"},
		},
		{
			DeviceClass:         "edge",
			MinSampleIntervalMs: defaultEdgeSampleMs,
			QueuePriority:       3,
			Roles:               []string{"cpe", "station", "client", "endpoint", "sensor", "iot"},
		},
		{
			DeviceClass:         "default",
			MinSampleIntervalMs: defaultGenericSampleMs,
			QueuePriority:       4,
			Roles:               []string{"device"},
		},
	}
}

func normalizeTelemetryGovernorRules(rules []TelemetryClassGovernorRule) []TelemetryClassGovernorRule {
	if len(rules) == 0 {
		return defaultTelemetryGovernorRules()
	}

	fallback := defaultTelemetryGovernorRules()
	defaultByClass := make(map[string]TelemetryClassGovernorRule, len(fallback))
	for _, rule := range fallback {
		defaultByClass[rule.DeviceClass] = rule
	}

	out := make([]TelemetryClassGovernorRule, 0, len(rules)+1)
	seenClasses := map[string]struct{}{}
	for _, rule := range rules {
		className := strings.ToLower(strings.TrimSpace(rule.DeviceClass))
		if className == "" {
			className = "default"
		}
		base := defaultByClass[className]
		if base.DeviceClass == "" {
			base = defaultByClass["default"]
		}

		normalized := TelemetryClassGovernorRule{
			DeviceClass:         className,
			MinSampleIntervalMs: rule.MinSampleIntervalMs,
			QueuePriority:       rule.QueuePriority,
		}
		if normalized.MinSampleIntervalMs <= 0 {
			normalized.MinSampleIntervalMs = base.MinSampleIntervalMs
		}
		if normalized.QueuePriority < 0 {
			normalized.QueuePriority = base.QueuePriority
		}
		for _, role := range rule.Roles {
			trimmed := strings.ToLower(strings.TrimSpace(role))
			if trimmed == "" {
				continue
			}
			normalized.Roles = appendUnique(normalized.Roles, trimmed)
		}
		if len(normalized.Roles) == 0 {
			normalized.Roles = append([]string(nil), base.Roles...)
		}

		if _, exists := seenClasses[normalized.DeviceClass]; exists {
			continue
		}
		seenClasses[normalized.DeviceClass] = struct{}{}
		out = append(out, normalized)
	}

	if _, ok := seenClasses["default"]; !ok {
		out = append(out, defaultByClass["default"])
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].QueuePriority < out[j].QueuePriority
	})
	return out
}

func telemetryRuleForRole(role string, rules []TelemetryClassGovernorRule) TelemetryClassGovernorRule {
	if len(rules) == 0 {
		rules = defaultTelemetryGovernorRules()
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	for _, rule := range rules {
		for _, candidate := range rule.Roles {
			if normalizedRole != "" && normalizedRole == candidate {
				return rule
			}
		}
	}

	for _, rule := range rules {
		if rule.DeviceClass == "default" {
			return rule
		}
	}
	return rules[len(rules)-1]
}

func isTransitionEventType(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "device_down", "offline", "device_up", "online":
		return true
	default:
		return false
	}
}

func shouldBypassSampling(eventType string, incomingOnline, currentOnline *bool, hasFactPayload bool) bool {
	if hasFactPayload {
		return true
	}
	if isTransitionEventType(eventType) {
		return true
	}
	if incomingOnline != nil && currentOnline != nil && *incomingOnline != *currentOnline {
		return true
	}
	return false
}

func normalizeTelemetryRetentionPolicy(policy TelemetryRetentionPolicy) TelemetryRetentionPolicy {
	if policy.HotRetentionMs <= 0 {
		policy.HotRetentionMs = defaultHotRetentionMs
	}
	if policy.WarmRetentionMs <= 0 {
		policy.WarmRetentionMs = defaultWarmRetentionMs
	}
	if policy.ColdRetentionMs <= 0 {
		policy.ColdRetentionMs = defaultColdRetentionMs
	}
	if policy.WarmRetentionMs <= policy.HotRetentionMs {
		policy.WarmRetentionMs = policy.HotRetentionMs + defaultWarmRetentionMs
	}
	if policy.ColdRetentionMs <= policy.WarmRetentionMs {
		policy.ColdRetentionMs = policy.WarmRetentionMs + defaultColdRetentionMs
	}
	if policy.HotMaxSamples <= 0 {
		policy.HotMaxSamples = defaultHotMaxSamples
	}
	if policy.WarmMaxSamples <= 0 {
		policy.WarmMaxSamples = defaultWarmMaxSamples
	}
	if policy.ColdMaxSamples <= 0 {
		policy.ColdMaxSamples = defaultColdMaxSamples
	}
	return policy
}

func trimTelemetrySamples(samples []TelemetrySample, maxSamples int) []TelemetrySample {
	if maxSamples <= 0 {
		return nil
	}
	if len(samples) <= maxSamples {
		return samples
	}
	return append([]TelemetrySample(nil), samples[len(samples)-maxSamples:]...)
}

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func normalizeTelemetrySample(sample TelemetrySample, nowMs int64) TelemetrySample {
	if sample.ObservedAt <= 0 {
		sample.ObservedAt = nowMs
	}
	if sample.SampleID == "" {
		sample.SampleID = "ts-" + randomID()
	}
	sample.DeviceID = strings.TrimSpace(sample.DeviceID)
	sample.IdentityID = strings.TrimSpace(sample.IdentityID)
	sample.Source = strings.TrimSpace(sample.Source)
	if sample.Source == "" {
		sample.Source = defaultDeviceSourceName
	}
	sample.EventType = strings.ToLower(strings.TrimSpace(sample.EventType))
	if sample.EventType == "" {
		sample.EventType = "telemetry"
	}
	sample.DeviceRole = strings.TrimSpace(sample.DeviceRole)
	sample.SiteID = strings.TrimSpace(sample.SiteID)
	sample.ObservedISO = time.UnixMilli(sample.ObservedAt).UTC().Format(time.RFC3339)
	sample.Online = cloneBoolPtr(sample.Online)
	sample.LatencyMs = cloneFloat64Ptr(sample.LatencyMs)
	return sample
}

func (s *Store) applyTelemetryRetentionLocked(nowMs int64) {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	s.TelemetryRetentionPolicy = normalizeTelemetryRetentionPolicy(s.TelemetryRetentionPolicy)
	policy := s.TelemetryRetentionPolicy

	hotCutoff := nowMs - policy.HotRetentionMs
	warmCutoff := nowMs - policy.WarmRetentionMs
	coldCutoff := nowMs - policy.ColdRetentionMs

	nextHot := make([]TelemetrySample, 0, len(s.TelemetryHot))
	promoteWarm := make([]TelemetrySample, 0, len(s.TelemetryHot))
	for _, raw := range s.TelemetryHot {
		sample := normalizeTelemetrySample(raw, nowMs)
		if sample.DeviceID == "" {
			continue
		}
		if sample.ObservedAt <= hotCutoff {
			promoteWarm = append(promoteWarm, sample)
			continue
		}
		nextHot = append(nextHot, sample)
	}

	warmCombined := make([]TelemetrySample, 0, len(s.TelemetryWarm)+len(promoteWarm))
	for _, raw := range s.TelemetryWarm {
		sample := normalizeTelemetrySample(raw, nowMs)
		if sample.DeviceID == "" {
			continue
		}
		warmCombined = append(warmCombined, sample)
	}
	warmCombined = append(warmCombined, promoteWarm...)

	nextWarm := make([]TelemetrySample, 0, len(warmCombined))
	promoteCold := make([]TelemetrySample, 0, len(warmCombined))
	for _, sample := range warmCombined {
		if sample.ObservedAt <= warmCutoff {
			promoteCold = append(promoteCold, sample)
			continue
		}
		nextWarm = append(nextWarm, sample)
	}

	coldCombined := make([]TelemetrySample, 0, len(s.TelemetryCold)+len(promoteCold))
	for _, raw := range s.TelemetryCold {
		sample := normalizeTelemetrySample(raw, nowMs)
		if sample.DeviceID == "" {
			continue
		}
		coldCombined = append(coldCombined, sample)
	}
	coldCombined = append(coldCombined, promoteCold...)

	nextCold := make([]TelemetrySample, 0, len(coldCombined))
	for _, sample := range coldCombined {
		if sample.ObservedAt <= coldCutoff {
			continue
		}
		nextCold = append(nextCold, sample)
	}

	s.TelemetryHot = trimTelemetrySamples(nextHot, policy.HotMaxSamples)
	s.TelemetryWarm = trimTelemetrySamples(nextWarm, policy.WarmMaxSamples)
	s.TelemetryCold = trimTelemetrySamples(nextCold, policy.ColdMaxSamples)
}

func (s *Store) backfillTelemetryRetentionFromObservationsLocked() {
	nowMs := time.Now().UnixMilli()
	for _, obs := range s.SourceObservations {
		deviceID := strings.TrimSpace(obs.DeviceID)
		if deviceID == "" {
			continue
		}
		eventType := "telemetry"
		if obs.Online != nil && !*obs.Online {
			eventType = "offline"
		}
		source := strings.TrimSpace(obs.Source)
		if source == "" {
			source = defaultDeviceSourceName
		}
		sample := TelemetrySample{
			SampleID:   "ts-" + randomID(),
			DeviceID:   deviceID,
			IdentityID: strings.TrimSpace(obs.IdentityID),
			Source:     source,
			EventType:  eventType,
			DeviceRole: strings.TrimSpace(obs.Role),
			SiteID:     strings.TrimSpace(obs.SiteID),
			Online:     cloneBoolPtr(obs.Online),
			LatencyMs:  cloneFloat64Ptr(obs.LatencyMs),
			ObservedAt: obs.ObservedAt,
		}
		if sample.ObservedAt <= 0 {
			sample.ObservedAt = nowMs
		}
		sample.ObservedISO = time.UnixMilli(sample.ObservedAt).UTC().Format(time.RFC3339)
		s.TelemetryHot = append(s.TelemetryHot, sample)
	}
}

func (s *Store) evaluateTelemetryIngestDecisionLocked(deviceID, deviceRole, eventType string, incomingOnline, currentOnline *bool, hasFactPayload bool, nowMs int64) TelemetryIngestDecision {
	rule := telemetryRuleForRole(deviceRole, s.TelemetryGovernorRules)
	decision := TelemetryIngestDecision{
		Accepted:            true,
		DeviceClass:         rule.DeviceClass,
		QueuePriority:       rule.QueuePriority,
		MinSampleIntervalMs: rule.MinSampleIntervalMs,
	}

	if s.TelemetryLastByDevice == nil {
		s.TelemetryLastByDevice = map[string]int64{}
	}

	if hasFactPayload {
		decision.Reason = "inventory_fact_payload"
		s.TelemetryLastByDevice[deviceID] = nowMs
		s.TelemetryAcceptedSamples++
		s.TelemetryGovernorLastEvalMs = nowMs
		s.pruneSamplingStateLocked(nowMs)
		return decision
	}

	if shouldBypassSampling(eventType, incomingOnline, currentOnline, false) {
		decision.Reason = "state_transition"
		s.TelemetryLastByDevice[deviceID] = nowMs
		s.TelemetryAcceptedSamples++
		s.TelemetryGovernorLastEvalMs = nowMs
		s.pruneSamplingStateLocked(nowMs)
		return decision
	}

	lastAcceptedAt, seen := s.TelemetryLastByDevice[deviceID]
	if seen && nowMs-lastAcceptedAt < rule.MinSampleIntervalMs {
		decision.Accepted = false
		decision.Reason = "sampled_by_class_interval"
		s.TelemetryDroppedSamples++
		s.TelemetryGovernorLastEvalMs = nowMs
		return decision
	}

	decision.Reason = "accepted"
	s.TelemetryLastByDevice[deviceID] = nowMs
	s.TelemetryAcceptedSamples++
	s.TelemetryGovernorLastEvalMs = nowMs
	s.pruneSamplingStateLocked(nowMs)
	return decision
}

func (s *Store) pruneSamplingStateLocked(nowMs int64) {
	if len(s.TelemetryLastByDevice) == 0 {
		return
	}
	cutoff := nowMs - int64((24*time.Hour)/time.Millisecond)
	for deviceID, seenAt := range s.TelemetryLastByDevice {
		if seenAt <= cutoff {
			delete(s.TelemetryLastByDevice, deviceID)
		}
	}
	if len(s.TelemetryLastByDevice) <= maxSamplingStateDevices {
		return
	}
	toDrop := len(s.TelemetryLastByDevice) - maxSamplingStateDevices
	for deviceID := range s.TelemetryLastByDevice {
		delete(s.TelemetryLastByDevice, deviceID)
		toDrop--
		if toDrop <= 0 {
			break
		}
	}
}

func (s *Store) applyTelemetryGapDetectionLocked(nowMs int64) (int, int, bool) {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	s.TelemetryGovernorLastEvalMs = nowMs
	if len(s.Devices) == 0 {
		return 0, 0, false
	}

	activeGapByDevice := map[string]int{}
	for i := range s.Incidents {
		if s.Incidents[i].Type != "telemetry_gap" || s.Incidents[i].Resolved != nil {
			continue
		}
		activeGapByDevice[s.Incidents[i].DeviceID] = i
	}

	created := 0
	resolved := 0
	changed := false
	nowISO := time.UnixMilli(nowMs).UTC().Format(time.RFC3339)
	for _, dev := range s.Devices {
		if dev.LastSeen <= 0 {
			continue
		}
		rule := telemetryRuleForRole(dev.Role, s.TelemetryGovernorRules)
		thresholdMs := rule.MinSampleIntervalMs * telemetryGapMultiplier
		if thresholdMs < minTelemetryGapMs {
			thresholdMs = minTelemetryGapMs
		}
		ageMs := nowMs - dev.LastSeen
		if ageMs > thresholdMs {
			if _, exists := activeGapByDevice[dev.ID]; !exists {
				inc := Incident{
					ID:       "inc-" + randomID(),
					DeviceID: dev.ID,
					Type:     "telemetry_gap",
					Severity: "warning",
					Started:  nowISO,
					Message:  "Missing telemetry signal beyond class threshold",
					Source:   "telemetry_gap_detector",
				}
				s.Incidents = append(s.Incidents, inc)
				created++
				changed = true
			}
			continue
		}

		if idx, exists := activeGapByDevice[dev.ID]; exists {
			s.Incidents[idx].Resolved = &nowISO
			resolved++
			changed = true
		}
	}
	return created, resolved, changed
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

func (s *Store) ListHAPairs(limit int, state string) ([]HAPairStatus, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	stateFilter := strings.ToLower(strings.TrimSpace(state))
	filtered := make([]HAPairStatus, 0, min(limit, len(s.HAPairs)))
	total := 0
	for _, pair := range s.HAPairs {
		if stateFilter != "" && strings.ToLower(strings.TrimSpace(pair.State)) != stateFilter {
			continue
		}
		total++
		if len(filtered) >= limit {
			continue
		}
		filtered = append(filtered, pair)
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
}

func (s *Store) ListHAFailoverEvents(limit int, pairID, eventType string) ([]HAFailoverEvent, bool, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	pairFilter := strings.TrimSpace(pairID)
	typeFilter := strings.ToLower(strings.TrimSpace(eventType))
	filtered := make([]HAFailoverEvent, 0, min(limit, len(s.HAFailoverEvents)))
	total := 0
	for i := len(s.HAFailoverEvents) - 1; i >= 0; i-- {
		event := s.HAFailoverEvents[i]
		if pairFilter != "" && event.PairID != pairFilter {
			continue
		}
		if typeFilter != "" && strings.ToLower(strings.TrimSpace(event.EventType)) != typeFilter {
			continue
		}
		total++
		if len(filtered) >= limit {
			continue
		}
		filtered = append(filtered, event)
	}
	truncated := total > len(filtered)
	return filtered, truncated, limit
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

type haPairCandidate struct {
	pairID    string
	identityA string
	identityB string
	siteID    string
	role      string
	score     int
}

func (s *Store) updateHAPairWatcherLocked(nowMs int64) {
	nextPairs := s.computeHAPairsLocked(nowMs)
	prevByPairID := make(map[string]HAPairStatus, len(s.HAPairs))
	for _, pair := range s.HAPairs {
		prevByPairID[pair.PairID] = pair
	}

	nowISO := time.UnixMilli(nowMs).UTC().Format(time.RFC3339)
	for i := range nextPairs {
		next := &nextPairs[i]
		prev, hadPrev := prevByPairID[next.PairID]
		if hadPrev {
			next.LastTransitionAt = prev.LastTransitionAt
			next.LastTransitionAtISO = prev.LastTransitionAtISO
		}

		stateChanged := !hadPrev || prev.State != next.State || prev.ActiveIdentityID != next.ActiveIdentityID
		if !stateChanged {
			continue
		}
		next.LastTransitionAt = nowMs
		next.LastTransitionAtISO = nowISO
		if !hadPrev {
			continue
		}

		eventType := "state_change"
		if next.State == "failover" && prev.State != "failover" {
			eventType = "failover"
		} else if prev.State == "failover" && next.State == "redundant" {
			eventType = "recovered"
		} else if prev.ActiveIdentityID != "" && next.ActiveIdentityID != "" && prev.ActiveIdentityID != next.ActiveIdentityID {
			eventType = "failover"
		}

		s.HAFailoverEvents = append(s.HAFailoverEvents, HAFailoverEvent{
			EventID:              "haevt-" + randomID(),
			PairID:               next.PairID,
			EventType:            eventType,
			FromState:            prev.State,
			ToState:              next.State,
			FromActiveIdentityID: prev.ActiveIdentityID,
			ToActiveIdentityID:   next.ActiveIdentityID,
			NodeAIdentityID:      next.NodeAIdentityID,
			NodeBIdentityID:      next.NodeBIdentityID,
			ObservedAt:           nowMs,
			ObservedAtISO:        nowISO,
			Message:              buildHAFailoverEventMessage(prev, *next, eventType),
		})
	}

	s.HAPairs = nextPairs
	if len(s.HAFailoverEvents) > maxHAFailoverEvents {
		s.HAFailoverEvents = append([]HAFailoverEvent(nil), s.HAFailoverEvents[len(s.HAFailoverEvents)-maxHAFailoverEvents:]...)
	}
}

func (s *Store) computeHAPairsLocked(nowMs int64) []HAPairStatus {
	identByID := make(map[string]DeviceIdentity, len(s.DeviceIdentities))
	groups := map[string][]string{}
	for _, ident := range s.DeviceIdentities {
		identityID := strings.TrimSpace(ident.IdentityID)
		site := strings.TrimSpace(ident.SiteID)
		role := strings.TrimSpace(ident.Role)
		if identityID == "" || site == "" || role == "" {
			continue
		}
		identByID[identityID] = ident
		groupKey := normalizeKeyToken(site) + "|" + normalizeKeyToken(role)
		groups[groupKey] = append(groups[groupKey], identityID)
	}

	candidates := map[string]haPairCandidate{}
	addCandidate := func(identityA, identityB, siteID, role string, score int) {
		a, b := haPairIdentityOrder(identityA, identityB)
		if a == "" || b == "" || a == b {
			return
		}
		pairID := topologyHAPairID(a, b)
		existing, ok := candidates[pairID]
		if !ok {
			candidates[pairID] = haPairCandidate{
				pairID:    pairID,
				identityA: a,
				identityB: b,
				siteID:    siteID,
				role:      role,
				score:     score,
			}
			return
		}
		existing.score += score
		if existing.siteID == "" {
			existing.siteID = siteID
		}
		if existing.role == "" {
			existing.role = role
		}
		candidates[pairID] = existing
	}

	for _, identityIDs := range groups {
		if len(identityIDs) != 2 {
			continue
		}
		identA := identByID[identityIDs[0]]
		identB := identByID[identityIDs[1]]
		addCandidate(identA.IdentityID, identB.IdentityID, identA.SiteID, identA.Role, 1)
	}

	nodes, edges, _ := s.buildTopologyGraphLocked()
	nodeIdentity := map[string]string{}
	for _, node := range nodes {
		if node.Kind != "managed" {
			continue
		}
		identityID := strings.TrimSpace(node.IdentityID)
		if identityID == "" {
			continue
		}
		nodeIdentity[node.NodeID] = identityID
	}
	for _, edge := range edges {
		identityA := nodeIdentity[edge.FromNodeID]
		identityB := nodeIdentity[edge.ToNodeID]
		if identityA == "" || identityB == "" || identityA == identityB {
			continue
		}
		identA, okA := identByID[identityA]
		identB, okB := identByID[identityB]
		if !okA || !okB {
			continue
		}
		if normalizeKeyToken(identA.SiteID) != normalizeKeyToken(identB.SiteID) {
			continue
		}
		if normalizeKeyToken(identA.Role) != normalizeKeyToken(identB.Role) {
			continue
		}
		addCandidate(identityA, identityB, identA.SiteID, identA.Role, 2)
	}

	candidateList := make([]haPairCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		candidateList = append(candidateList, candidate)
	}
	sort.Slice(candidateList, func(i, j int) bool {
		if candidateList[i].score != candidateList[j].score {
			return candidateList[i].score > candidateList[j].score
		}
		return candidateList[i].pairID < candidateList[j].pairID
	})

	used := map[string]struct{}{}
	pairs := make([]HAPairStatus, 0, len(candidateList))
	nowISO := time.UnixMilli(nowMs).UTC().Format(time.RFC3339)
	for _, candidate := range candidateList {
		if _, ok := used[candidate.identityA]; ok {
			continue
		}
		if _, ok := used[candidate.identityB]; ok {
			continue
		}
		identA, okA := identByID[candidate.identityA]
		identB, okB := identByID[candidate.identityB]
		if !okA || !okB {
			continue
		}

		onlineA, tsA, onlineB, tsB, observedSample := s.resolveHAPairOnlineStatesLocked(identA, identB)
		state, activeID, standbyID := evaluateHAPairState(identA.IdentityID, identB.IdentityID, onlineA, onlineB, tsA, tsB)
		pairs = append(pairs, HAPairStatus{
			PairID:               candidate.pairID,
			SiteID:               candidate.siteID,
			Role:                 candidate.role,
			NodeAIdentityID:      identA.IdentityID,
			NodeAName:            identA.Name,
			NodeAOnline:          onlineA,
			NodeBIdentityID:      identB.IdentityID,
			NodeBName:            identB.Name,
			NodeBOnline:          onlineB,
			State:                state,
			ActiveIdentityID:     activeID,
			StandbyIdentityID:    standbyID,
			LastEvaluatedAt:      nowMs,
			LastEvaluatedAtISO:   nowISO,
			ObservedSourceSample: observedSample,
		})
		used[candidate.identityA] = struct{}{}
		used[candidate.identityB] = struct{}{}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].SiteID != pairs[j].SiteID {
			return pairs[i].SiteID < pairs[j].SiteID
		}
		if pairs[i].Role != pairs[j].Role {
			return pairs[i].Role < pairs[j].Role
		}
		return pairs[i].PairID < pairs[j].PairID
	})
	return pairs
}

func (s *Store) resolveHAPairOnlineStatesLocked(identityA, identityB DeviceIdentity) (*bool, int64, *bool, int64, int64) {
	onlineA, tsA := s.latestIdentityOnlineStateLocked(identityA)
	onlineB, tsB := s.latestIdentityOnlineStateLocked(identityB)
	return onlineA, tsA, onlineB, tsB, max(tsA, tsB)
}

func (s *Store) latestIdentityOnlineStateLocked(identity DeviceIdentity) (*bool, int64) {
	identityID := strings.TrimSpace(identity.IdentityID)
	for i := len(s.SourceObservations) - 1; i >= 0; i-- {
		obs := s.SourceObservations[i]
		if obs.IdentityID != identityID || obs.Online == nil {
			continue
		}
		v := *obs.Online
		return &v, obs.ObservedAt
	}

	primaryDeviceID := strings.TrimSpace(identity.PrimaryDeviceID)
	if primaryDeviceID == "" {
		return nil, 0
	}
	for _, device := range s.Devices {
		if strings.TrimSpace(device.ID) != primaryDeviceID {
			continue
		}
		v := device.Online
		return &v, device.LastSeen
	}
	return nil, 0
}

func evaluateHAPairState(identityAID, identityBID string, onlineA, onlineB *bool, tsA, tsB int64) (string, string, string) {
	aKnown := onlineA != nil
	bKnown := onlineB != nil
	aUp := aKnown && *onlineA
	bUp := bKnown && *onlineB

	switch {
	case aUp && bUp:
		if tsB > tsA {
			return "redundant", identityBID, identityAID
		}
		return "redundant", identityAID, identityBID
	case aKnown && bKnown && aUp && !bUp:
		return "failover", identityAID, identityBID
	case aKnown && bKnown && !aUp && bUp:
		return "failover", identityBID, identityAID
	case aKnown && bKnown && !aUp && !bUp:
		return "down", "", ""
	case aUp && !bKnown:
		return "unknown", identityAID, identityBID
	case bUp && !aKnown:
		return "unknown", identityBID, identityAID
	default:
		return "unknown", "", ""
	}
}

func buildHAFailoverEventMessage(prev, next HAPairStatus, eventType string) string {
	nodeA := firstNonEmpty(next.NodeAName, next.NodeAIdentityID)
	nodeB := firstNonEmpty(next.NodeBName, next.NodeBIdentityID)
	switch eventType {
	case "failover":
		return "HA failover detected for " + nodeA + " / " + nodeB
	case "recovered":
		return "HA pair recovered for " + nodeA + " / " + nodeB
	default:
		return "HA state changed for " + nodeA + " / " + nodeB
	}
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
	s.updateHAPairWatcherLocked(time.Now().UnixMilli())
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
	device, incident, _, ok := s.IngestTelemetryWithDecision(req)
	return device, incident, ok
}

func (s *Store) IngestTelemetryWithDecision(req TelemetryIngestRequest) (Device, *Incident, TelemetryIngestDecision, bool) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return Device{}, nil, TelemetryIngestDecision{}, false
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
	var existingOnline *bool
	for i := range s.Devices {
		if s.Devices[i].ID == deviceID {
			idx = i
			value := s.Devices[i].Online
			existingOnline = &value
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

	hasFactPayload := len(req.Interfaces) > 0 || len(req.Neighbors) > 0
	decision := s.evaluateTelemetryIngestDecisionLocked(deviceID, deviceRole, eventType, req.Online, existingOnline, hasFactPayload, nowMs)
	if !decision.Accepted {
		s.updateHAPairWatcherLocked(nowMs)
		s.applyTelemetryGapDetectionLocked(nowMs)
		deviceCopy := s.Devices[idx]
		s.mu.Unlock()
		s.save()
		return deviceCopy, nil, decision, true
	}

	onlineState := online
	identityID := s.upsertIdentityFromTelemetryLocked(req, source, deviceName, deviceRole, siteID, nowMs, &onlineState)
	s.appendTelemetrySampleLocked(req, source, deviceID, identityID, deviceRole, siteID, onlineState, nowMs)
	s.applyTelemetryRetentionLocked(nowMs)

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

	s.updateHAPairWatcherLocked(nowMs)
	s.applyTelemetryGapDetectionLocked(nowMs)
	deviceCopy := s.Devices[idx]
	s.mu.Unlock()

	s.save()
	return deviceCopy, created, decision, true
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

func (s *Store) appendTelemetrySampleLocked(req TelemetryIngestRequest, source, deviceID, identityID, deviceRole, siteID string, online bool, nowMs int64) {
	sample := TelemetrySample{
		SampleID:   "ts-" + randomID(),
		DeviceID:   strings.TrimSpace(deviceID),
		IdentityID: strings.TrimSpace(identityID),
		Source:     strings.TrimSpace(source),
		EventType:  strings.ToLower(strings.TrimSpace(req.EventType)),
		DeviceRole: strings.TrimSpace(deviceRole),
		SiteID:     strings.TrimSpace(siteID),
		Online:     &online,
		ObservedAt: nowMs,
	}
	if sample.EventType == "" {
		sample.EventType = "telemetry"
	}
	if sample.Source == "" {
		sample.Source = defaultDeviceSourceName
	}
	if req.Online != nil {
		sample.Online = cloneBoolPtr(req.Online)
	}
	if req.LatencyMs != nil {
		sample.LatencyMs = cloneFloat64Ptr(req.LatencyMs)
	}
	sample.ObservedISO = time.UnixMilli(nowMs).UTC().Format(time.RFC3339)
	s.TelemetryHot = append(s.TelemetryHot, sample)
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
		_ = s.upsertIdentityFromTelemetryLocked(req, source, dev.Name, dev.Role, dev.SiteID, nowMs, &online)
	}
}

func (s *Store) upsertIdentityFromTelemetryLocked(req TelemetryIngestRequest, source, deviceName, deviceRole, siteID string, nowMs int64, online *bool) string {
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
		return ""
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
	s.retentionLast = s.applyRetentionPolicyLocked(nowMs)

	for _, key := range identityKeysFromObservation(obs) {
		s.identityIndex[key] = identity.IdentityID
	}
	return identity.IdentityID
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

func haPairIdentityOrder(identityA, identityB string) (string, string) {
	a := strings.TrimSpace(identityA)
	b := strings.TrimSpace(identityB)
	if a <= b {
		return a, b
	}
	return b, a
}

func topologyHAPairID(identityA, identityB string) string {
	a, b := haPairIdentityOrder(identityA, identityB)
	if a == "" || b == "" {
		return ""
	}
	return "ha:" + normalizeKeyToken(a+"|"+b)
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
