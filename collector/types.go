package collector

import (
	"time"

	"neurolink/apex-server-monitor/statusapi"
)

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

type Target struct {
	Cluster ClusterID
	Name    string
	Address string
}

type Config struct {
	Provider             statusapi.Provider
	PollInterval         time.Duration
	ProcessCheckInterval time.Duration
}

type PingConfig struct {
	PingTimeout        time.Duration
	BattlePingInterval time.Duration
	IdlePingInterval   time.Duration
}

type ModeState struct {
	Mode        Mode
	BattleMode  bool
	ProcessName string
	CheckedAt   time.Time
}

type Snapshot struct {
	Mode        Mode
	BattleMode  bool
	ProcessName string
	ModeChecked time.Time
	Status      statusapi.Snapshot
	UpdatedAt   time.Time
	LastError   string
}

type PollResult struct {
	Snapshot statusapi.Snapshot
	Err      error
	At       time.Time
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

func DefaultConfig() Config {
	return Config{
		Provider:             statusapi.NewDemoProvider(),
		PollInterval:         time.Minute,
		ProcessCheckInterval: 5 * time.Second,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.Provider == nil {
		c.Provider = defaults.Provider
	}
	if c.PollInterval <= 0 {
		c.PollInterval = defaults.PollInterval
	}
	if c.ProcessCheckInterval <= 0 {
		c.ProcessCheckInterval = defaults.ProcessCheckInterval
	}
	return c
}

func defaultPingConfig() PingConfig {
	return PingConfig{
		PingTimeout:        2 * time.Second,
		BattlePingInterval: time.Second,
		IdlePingInterval:   15 * time.Second,
	}
}
