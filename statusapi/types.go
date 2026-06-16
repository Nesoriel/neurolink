package statusapi

import (
	"context"
	"time"
)

const Attribution = "Data from apexlegendsstatus.com"

type Provider interface {
	Fetch(ctx context.Context) (Snapshot, error)
}

type SourceMode string

const (
	SourceLive SourceMode = "LIVE"
	SourceDemo SourceMode = "DEMO"
)

type ServiceID string

const (
	ServiceCrossplayAuth ServiceID = "crossplay-auth"
	ServiceMatchmaking   ServiceID = "lobby-matchmaking"
	ServicePCLogin       ServiceID = "pc-login"
	ServicePlayerAccount ServiceID = "player-accounts"
	ServiceAPIHealth     ServiceID = "api-health"
)

type Status string

const (
	StatusHealthy  Status = "HEALTHY"
	StatusDegraded Status = "DEGRADED"
	StatusDown     Status = "DOWN"
	StatusUnknown  Status = "UNKNOWN"
)

type RegionStatus struct {
	Name          string
	Status        Status
	Label         string
	Latency       time.Duration
	HasLatency    bool
	UptimePercent float64
	HasUptime     bool
	CheckedAt     time.Time
}

type ServiceStatus struct {
	ID        ServiceID
	Name      string
	Status    Status
	Summary   string
	UpdatedAt time.Time
	Regions   []RegionStatus
}

type RecentReport struct {
	Country   string
	At        string
	Issue     string
	Platform  string
	ErrorCode string
}

type Snapshot struct {
	Source        SourceMode
	Attribution   string
	GeneratedAt   time.Time
	Overall       Status
	Services      []ServiceStatus
	RecentReports []RecentReport
	Notice        string
	LastError     string
}

func (s Status) IsIncident() bool {
	return s == StatusDegraded || s == StatusDown
}

func (s Status) Weight() int {
	switch s {
	case StatusDown:
		return 4
	case StatusDegraded:
		return 3
	case StatusUnknown:
		return 2
	case StatusHealthy:
		return 1
	default:
		return 2
	}
}

func CoreServices() []ServiceStatus {
	return []ServiceStatus{
		{ID: ServiceCrossplayAuth, Name: "Crossplay Auth", Status: StatusUnknown, Summary: "No status data yet"},
		{ID: ServiceMatchmaking, Name: "Lobby / Matchmaking", Status: StatusUnknown, Summary: "No status data yet"},
		{ID: ServicePCLogin, Name: "PC / Desktop Logins", Status: StatusUnknown, Summary: "No status data yet"},
		{ID: ServicePlayerAccount, Name: "Player Accounts", Status: StatusUnknown, Summary: "No status data yet"},
		{ID: ServiceAPIHealth, Name: "Apex Legends Status API", Status: StatusUnknown, Summary: "No status data yet"},
	}
}

func ServiceByID(snapshot Snapshot, id ServiceID) (ServiceStatus, bool) {
	for _, service := range snapshot.Services {
		if service.ID == id {
			return service, true
		}
	}
	return ServiceStatus{}, false
}
