package collector

import "time"

type Mode string

const (
	ModeIdle   Mode = "IDLE"
	ModeBattle Mode = "BATTLE"
)

type ClusterID string

const (
	ClusterHongKong  ClusterID = "hong-kong"
	ClusterSingapore ClusterID = "singapore"
)

type Status string

const (
	StatusOnline   Status = "ONLINE"
	StatusHighLoss Status = "HIGH LOSS"
	StatusOffline  Status = "OFFLINE"
)

const (
	// These RFC 5737 TEST-NET placeholders are intentionally safe and should not
	// route on the public internet. Replace them with real Apex endpoint IPs when
	// you are ready to monitor live targets.
	HongKongTargetIP  = "192.0.2.10"
	SingaporeTargetIP = "198.51.100.10"
)

type Target struct {
	Cluster ClusterID
	Name    string
	Address string
}

type Config struct {
	Targets                 []Target
	PingTimeout             time.Duration
	BattlePingInterval      time.Duration
	IdlePingInterval        time.Duration
	ProcessCheckInterval    time.Duration
	WindowSize              int
	LossThresholdPercent    float64
	LossNotificationSamples int
}

type ModeState struct {
	Mode        Mode
	BattleMode  bool
	ProcessName string
	CheckedAt   time.Time
}

type PingSample struct {
	Cluster ClusterID
	Target  string
	Address string
	Mode    Mode
	At      time.Time
	RTT     time.Duration
	Success bool
	Error   string
}

type ClusterMetrics struct {
	Cluster       ClusterID
	Target        string
	Address       string
	Status        Status
	AvgLatency    time.Duration
	PacketLoss    float64
	Jitter        time.Duration
	SignalBars    int
	WindowSize    int
	SampleCount   int
	LastUpdated   time.Time
	LastError     string
	LastRTT       time.Duration
	LastSucceeded bool
}

type MetricsSnapshot struct {
	Mode        Mode
	BattleMode  bool
	ProcessName string
	UpdatedAt   time.Time
	Clusters    map[ClusterID]ClusterMetrics
}

func DefaultConfig() Config {
	return Config{
		Targets: []Target{
			{Cluster: ClusterHongKong, Name: "Hong Kong Server Cluster", Address: HongKongTargetIP},
			{Cluster: ClusterSingapore, Name: "Singapore Server Cluster", Address: SingaporeTargetIP},
		},
		PingTimeout:             2 * time.Second,
		BattlePingInterval:      1 * time.Second,
		IdlePingInterval:        15 * time.Second,
		ProcessCheckInterval:    5 * time.Second,
		WindowSize:              10,
		LossThresholdPercent:    5,
		LossNotificationSamples: 3,
	}
}

func (c Config) withDefaults() Config {
	if len(c.Targets) == 0 {
		c.Targets = DefaultConfig().Targets
	}
	if c.PingTimeout <= 0 {
		c.PingTimeout = 2 * time.Second
	}
	if c.BattlePingInterval <= 0 {
		c.BattlePingInterval = time.Second
	}
	if c.IdlePingInterval <= 0 {
		c.IdlePingInterval = 15 * time.Second
	}
	if c.ProcessCheckInterval <= 0 {
		c.ProcessCheckInterval = 5 * time.Second
	}
	if c.WindowSize <= 0 {
		c.WindowSize = 10
	}
	if c.LossThresholdPercent <= 0 {
		c.LossThresholdPercent = 5
	}
	if c.LossNotificationSamples <= 0 {
		c.LossNotificationSamples = 3
	}
	return c
}
