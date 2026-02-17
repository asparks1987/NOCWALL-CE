package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu         sync.RWMutex
	Devices    []Device              `json:"devices"`
	Incidents  []Incident            `json:"incidents"`
	Agents     []Agent               `json:"agents"`
	PushTokens []PushRegisterRequest `json:"push_tokens"`
	Users      []User                `json:"users"`

	filePath string
}

type storePersist struct {
	Devices    []Device              `json:"devices"`
	Incidents  []Incident            `json:"incidents"`
	Agents     []Agent               `json:"agents"`
	PushTokens []PushRegisterRequest `json:"push_tokens"`
	Users      []User                `json:"users"`
}

func LoadStore(path string) *Store {
	s := &Store{
		Devices:   seedDevices(),
		Incidents: seedIncidents(),
		Users:     []User{{Username: "admin", Password: "admin"}},
		filePath:  path,
	}
	if path == "" {
		return s
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, s)
	}
	return s
}

func (s *Store) save() {
	if s.filePath == "" {
		return
	}

	s.mu.RLock()
	payload := storePersist{
		Devices:    append([]Device(nil), s.Devices...),
		Incidents:  append([]Incident(nil), s.Incidents...),
		Agents:     append([]Agent(nil), s.Agents...),
		PushTokens: append([]PushRegisterRequest(nil), s.PushTokens...),
		Users:      append([]User(nil), s.Users...),
	}
	s.mu.RUnlock()

	b, _ := json.MarshalIndent(payload, "", "  ")
	_ = os.WriteFile(s.filePath, b, 0o644)
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
		source = "ingest"
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


