package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	mu         sync.RWMutex
	Devices    []Device        `json:"devices"`
	Incidents  []Incident      `json:"incidents"`
	PushTokens []PushRegisterRequest `json:"push_tokens"`
	Users      []User          `json:"users"`

	filePath string
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
	defer s.mu.RUnlock()
	b, _ := json.MarshalIndent(s, "", "  ")
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

func (s *Store) AckIncident(id string, minutes int) (Incident, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Incidents {
		if s.Incidents[i].ID == id {
			until := time.Now().Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339)
			s.Incidents[i].AckUntil = &until
			s.save()
			return s.Incidents[i], true
		}
	}
	return Incident{}, false
}

func (s *Store) RegisterPush(req PushRegisterRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PushTokens = append(s.PushTokens, req)
	s.save()
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
