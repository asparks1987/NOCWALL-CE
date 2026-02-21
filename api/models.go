package main

import "time"

type MobileConfig struct {
	UispBaseURL  string          `json:"uisp_base_url"`
	APIBaseURL   string          `json:"api_base_url"`
	FeatureFlags map[string]bool `json:"feature_flags"`
	PushRegister string          `json:"push_register_url"`
	Environment  string          `json:"environment"`
	Version      string          `json:"version"`
	Banner       string          `json:"banner"`
}

type Device struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	SiteID    string   `json:"site_id"`
	Online    bool     `json:"online"`
	LatencyMs *float64 `json:"latency_ms"`
	AckUntil  *int64   `json:"ack_until"`
	Source    string   `json:"source,omitempty"`
	LastSeen  int64    `json:"last_seen,omitempty"`
}

type Incident struct {
	ID       string  `json:"id"`
	DeviceID string  `json:"device_id"`
	Type     string  `json:"type"`
	Severity string  `json:"severity"`
	Started  string  `json:"started_at"`
	Resolved *string `json:"resolved_at"`
	AckUntil *string `json:"ack_until"`
	Message  string  `json:"message,omitempty"`
	Source   string  `json:"source,omitempty"`
}

type Agent struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SiteID       string   `json:"site_id,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	LastSeen     int64    `json:"last_seen"`
	Status       string   `json:"status"`
}

type AgentRegisterRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	SiteID       string   `json:"site_id,omitempty"`
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type TelemetryIngestRequest struct {
	Source     string                   `json:"source,omitempty"`
	AgentID    string                   `json:"agent_id,omitempty"`
	EventType  string                   `json:"event_type,omitempty"`
	DeviceID   string                   `json:"device_id"`
	Device     string                   `json:"device,omitempty"`
	Hostname   string                   `json:"hostname,omitempty"`
	Mac        string                   `json:"mac,omitempty"`
	Serial     string                   `json:"serial,omitempty"`
	Model      string                   `json:"model,omitempty"`
	Vendor     string                   `json:"vendor,omitempty"`
	Role       string                   `json:"role,omitempty"`
	SiteID     string                   `json:"site_id,omitempty"`
	Online     *bool                    `json:"online,omitempty"`
	LatencyMs  *float64                 `json:"latency_ms,omitempty"`
	Message    string                   `json:"message,omitempty"`
	Interfaces []TelemetryInterfaceFact `json:"interfaces,omitempty"`
	Neighbors  []TelemetryNeighborFact  `json:"neighbors,omitempty"`
}

type TelemetryInterfaceFact struct {
	Name      string   `json:"name"`
	AdminUp   *bool    `json:"admin_up,omitempty"`
	OperUp    *bool    `json:"oper_up,omitempty"`
	RxBps     *float64 `json:"rx_bps,omitempty"`
	TxBps     *float64 `json:"tx_bps,omitempty"`
	ErrorRate *float64 `json:"error_rate,omitempty"`
}

type TelemetryNeighborFact struct {
	LocalInterface       string `json:"local_interface,omitempty"`
	NeighborIdentityHint string `json:"neighbor_identity_hint,omitempty"`
	NeighborDeviceName   string `json:"neighbor_device_name,omitempty"`
	NeighborInterface    string `json:"neighbor_interface,omitempty"`
	Protocol             string `json:"protocol,omitempty"`
}

type DeviceIdentity struct {
	IdentityID      string   `json:"identity_id"`
	PrimaryDeviceID string   `json:"primary_device_id"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	SiteID          string   `json:"site_id"`
	Hostname        string   `json:"hostname,omitempty"`
	MacAddress      string   `json:"mac_address,omitempty"`
	SerialNumber    string   `json:"serial_number,omitempty"`
	Vendor          string   `json:"vendor,omitempty"`
	Model           string   `json:"model,omitempty"`
	SourceRefs      []string `json:"source_refs,omitempty"`
	LastSeen        int64    `json:"last_seen"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type DeviceInterface struct {
	ID         string   `json:"id"`
	IdentityID string   `json:"identity_id"`
	Name       string   `json:"name"`
	AdminUp    *bool    `json:"admin_up,omitempty"`
	OperUp     *bool    `json:"oper_up,omitempty"`
	RxBps      *float64 `json:"rx_bps,omitempty"`
	TxBps      *float64 `json:"tx_bps,omitempty"`
	ErrorRate  *float64 `json:"error_rate,omitempty"`
	Source     string   `json:"source,omitempty"`
	UpdatedAt  string   `json:"updated_at"`
}

type NeighborLink struct {
	ID                    string `json:"id"`
	IdentityID            string `json:"identity_id"`
	LocalInterface        string `json:"local_interface,omitempty"`
	NeighborIdentityHint  string `json:"neighbor_identity_hint,omitempty"`
	NeighborDeviceName    string `json:"neighbor_device_name,omitempty"`
	NeighborInterfaceHint string `json:"neighbor_interface_hint,omitempty"`
	Protocol              string `json:"protocol,omitempty"`
	Source                string `json:"source,omitempty"`
	UpdatedAt             string `json:"updated_at"`
}

type HardwareProfile struct {
	IdentityID      string `json:"identity_id"`
	Vendor          string `json:"vendor,omitempty"`
	Model           string `json:"model,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	HardwareRev     string `json:"hardware_revision,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}

type SourceObservation struct {
	ObservationID string   `json:"observation_id"`
	IdentityID    string   `json:"identity_id"`
	Source        string   `json:"source"`
	DeviceID      string   `json:"device_id"`
	Name          string   `json:"name,omitempty"`
	Role          string   `json:"role,omitempty"`
	SiteID        string   `json:"site_id,omitempty"`
	Hostname      string   `json:"hostname,omitempty"`
	MacAddress    string   `json:"mac_address,omitempty"`
	SerialNumber  string   `json:"serial_number,omitempty"`
	Vendor        string   `json:"vendor,omitempty"`
	Model         string   `json:"model,omitempty"`
	Online        *bool    `json:"online,omitempty"`
	LatencyMs     *float64 `json:"latency_ms,omitempty"`
	ObservedAt    int64    `json:"observed_at"`
}

type DriftSnapshot struct {
	SnapshotID    string            `json:"snapshot_id"`
	IdentityID    string            `json:"identity_id"`
	Fingerprint   string            `json:"fingerprint"`
	Changed       bool              `json:"changed"`
	ObservedAt    int64             `json:"observed_at"`
	ObservedAtISO string            `json:"observed_at_iso"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

type InventorySchemaResponse struct {
	Version         int               `json:"version"`
	DeviceIdentity  []string          `json:"device_identity_fields"`
	DeviceInterface []string          `json:"device_interface_fields"`
	NeighborLink    []string          `json:"neighbor_link_fields"`
	HardwareProfile []string          `json:"hardware_profile_fields"`
	Observation     []string          `json:"source_observation_fields"`
	Notes           map[string]string `json:"notes,omitempty"`
}

type InventoryDriftResponse struct {
	LastUpdated int64           `json:"last_updated"`
	Count       int             `json:"count"`
	Snapshots   []DriftSnapshot `json:"snapshots"`
	Truncated   bool            `json:"truncated"`
	Limit       int             `json:"limit"`
	Stub        bool            `json:"stub"`
}

type InventoryIdentitiesResponse struct {
	LastUpdated int64            `json:"last_updated"`
	Count       int              `json:"count"`
	Identities  []DeviceIdentity `json:"identities"`
	Stub        bool             `json:"stub"`
}

type InventoryObservationsResponse struct {
	LastUpdated  int64               `json:"last_updated"`
	Count        int                 `json:"count"`
	Observations []SourceObservation `json:"observations"`
	Truncated    bool                `json:"truncated"`
	Limit        int                 `json:"limit"`
	Stub         bool                `json:"stub"`
}

type InventoryInterfacesResponse struct {
	LastUpdated int64             `json:"last_updated"`
	Count       int               `json:"count"`
	Interfaces  []DeviceInterface `json:"interfaces"`
	Truncated   bool              `json:"truncated"`
	Limit       int               `json:"limit"`
	Stub        bool              `json:"stub"`
}

type InventoryNeighborsResponse struct {
	LastUpdated int64          `json:"last_updated"`
	Count       int            `json:"count"`
	Neighbors   []NeighborLink `json:"neighbors"`
	Truncated   bool           `json:"truncated"`
	Limit       int            `json:"limit"`
	Stub        bool           `json:"stub"`
}

type LifecycleScore struct {
	IdentityID string   `json:"identity_id"`
	Score      int      `json:"score"`
	Level      string   `json:"level"`
	Reasons    []string `json:"reasons,omitempty"`
}

type InventoryLifecycleResponse struct {
	LastUpdated int64            `json:"last_updated"`
	Count       int              `json:"count"`
	Scores      []LifecycleScore `json:"scores"`
	Truncated   bool             `json:"truncated"`
	Limit       int              `json:"limit"`
	Stub        bool             `json:"stub"`
}

type TopologyNode struct {
	NodeID          string `json:"node_id"`
	IdentityID      string `json:"identity_id,omitempty"`
	Label           string `json:"label"`
	Role            string `json:"role,omitempty"`
	SiteID          string `json:"site_id,omitempty"`
	LastSeen        int64  `json:"last_seen,omitempty"`
	Kind            string `json:"kind"` // managed | external
	SourceRefsCount int    `json:"source_refs_count,omitempty"`
}

type TopologyEdge struct {
	EdgeID             string `json:"edge_id"`
	FromNodeID         string `json:"from_node_id"`
	ToNodeID           string `json:"to_node_id"`
	SourceIdentityID   string `json:"source_identity_id,omitempty"`
	TargetIdentityHint string `json:"target_identity_hint,omitempty"`
	LocalInterface     string `json:"local_interface,omitempty"`
	NeighborInterface  string `json:"neighbor_interface,omitempty"`
	Protocol           string `json:"protocol,omitempty"`
	Source             string `json:"source,omitempty"`
	UpdatedAt          string `json:"updated_at,omitempty"`
	Resolved           bool   `json:"resolved"`
}

type TopologyHealth struct {
	NodeCount            int `json:"node_count"`
	ManagedNodeCount     int `json:"managed_node_count"`
	EdgeCount            int `json:"edge_count"`
	UnknownNeighborEdges int `json:"unknown_neighbor_edges"`
	IsolatedManagedNodes int `json:"isolated_managed_nodes"`
	StaleManagedNodes24h int `json:"stale_managed_nodes_24h"`
	ConnectedComponents  int `json:"connected_components"`
}

type TopologyNodesResponse struct {
	LastUpdated int64          `json:"last_updated"`
	Count       int            `json:"count"`
	Nodes       []TopologyNode `json:"nodes"`
	Truncated   bool           `json:"truncated"`
	Limit       int            `json:"limit"`
	Stub        bool           `json:"stub"`
}

type TopologyEdgesResponse struct {
	LastUpdated int64          `json:"last_updated"`
	Count       int            `json:"count"`
	Edges       []TopologyEdge `json:"edges"`
	Truncated   bool           `json:"truncated"`
	Limit       int            `json:"limit"`
	Stub        bool           `json:"stub"`
}

type TopologyHealthResponse struct {
	LastUpdated int64          `json:"last_updated"`
	Health      TopologyHealth `json:"health"`
	Stub        bool           `json:"stub"`
}

type TopologyPathResponse struct {
	LastUpdated      int64          `json:"last_updated"`
	Found            bool           `json:"found"`
	SourceNodeID     string         `json:"source_node_id,omitempty"`
	TargetNodeID     string         `json:"target_node_id,omitempty"`
	SourceIdentityID string         `json:"source_identity_id,omitempty"`
	TargetIdentityID string         `json:"target_identity_id,omitempty"`
	Hops             int            `json:"hops"`
	Nodes            []TopologyNode `json:"nodes,omitempty"`
	Edges            []TopologyEdge `json:"edges,omitempty"`
	Message          string         `json:"message,omitempty"`
	Stub             bool           `json:"stub"`
}

type IdentityMergeRequest struct {
	PrimaryID    string   `json:"primary_id"`
	SecondaryID  string   `json:"secondary_id,omitempty"`
	SecondaryIDs []string `json:"secondary_ids,omitempty"`
}

type IdentityMergeResponse struct {
	OK      bool           `json:"ok"`
	Primary DeviceIdentity `json:"primary"`
	Merged  []string       `json:"merged"`
	Stub    bool           `json:"stub"`
	Message string         `json:"message,omitempty"`
}

type EventIngestRequest struct {
	Type     string `json:"type"`
	DeviceID string `json:"device_id"`
	Device   string `json:"device,omitempty"`
	Site     string `json:"site,omitempty"`
	Message  string `json:"message,omitempty"`
}

type TelemetryIngestResponse struct {
	Accepted bool      `json:"accepted"`
	Device   Device    `json:"device"`
	Incident *Incident `json:"incident,omitempty"`
	Stub     bool      `json:"stub"`
}

type SourcePollRequest struct {
	Cursor  string `json:"cursor,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Demo    bool   `json:"demo,omitempty"`
	Retries int    `json:"retries,omitempty"`
}

type SourcePollResponse struct {
	Source           string `json:"source"`
	Cursor           string `json:"cursor"`
	Fetched          int    `json:"fetched"`
	Normalized       int    `json:"normalized"`
	Emitted          int    `json:"emitted"`
	Deduped          int    `json:"deduped"`
	Ingested         int    `json:"ingested"`
	IncidentsCreated int    `json:"incidents_created"`
	Backfill         bool   `json:"backfill"`
	Demo             bool   `json:"demo"`
	DurationMs       int64  `json:"duration_ms"`
	Stub             bool   `json:"stub"`
	Error            string `json:"error,omitempty"`
}

type SourceStatus struct {
	Source         string `json:"source"`
	LastPollAt     string `json:"last_poll_at,omitempty"`
	LastCursor     string `json:"last_cursor,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	LastFetched    int    `json:"last_fetched"`
	LastNormalized int    `json:"last_normalized"`
	LastEmitted    int    `json:"last_emitted"`
	Demo           bool   `json:"demo"`
	Stub           bool   `json:"stub"`
}

type PushRegisterRequest struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	AppVersion string `json:"app_version"`
	Locale     string `json:"locale"`
}

type PushRegisterResponse struct {
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
}

type DevicesResponse struct {
	LastUpdated int64    `json:"last_updated"`
	Devices     []Device `json:"devices"`
}

type AckRequest struct {
	DurationMinutes int `json:"duration_minutes"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func seedDevices() []Device {
	lat12 := 12.0
	lat3 := 3.0
	now := time.Now().UnixMilli()
	return []Device{
		{ID: "gw-1", Name: "Gateway-1", Role: "gateway", SiteID: "site-1", Online: true, LatencyMs: &lat12, Source: "seed", LastSeen: now},
		{ID: "ap-1", Name: "AP-1", Role: "ap", SiteID: "site-1", Online: false, LatencyMs: nil, Source: "seed", LastSeen: now},
		{ID: "sw-1", Name: "Switch-1", Role: "switch", SiteID: "site-1", Online: true, LatencyMs: &lat3, Source: "seed", LastSeen: now},
	}
}

func seedIncidents() []Incident {
	now := time.Now().UTC().Format(time.RFC3339)
	return []Incident{
		{ID: "inc-1", DeviceID: "ap-1", Type: "offline", Severity: "critical", Started: now, Source: "seed"},
	}
}
