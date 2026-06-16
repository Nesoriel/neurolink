package main

import (
	"context"
	"testing"
	"time"

	"neurolink/apex-server-monitor/statusapi"
)

func TestParseAppConfigUsesEnvAPIKey(t *testing.T) {
	cfg, err := parseAppConfig(nil, func(key string) string {
		if key == envAPIKey {
			return "env-key"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("API key = %q, want env-key", cfg.APIKey)
	}
	if cfg.PollInterval != time.Minute {
		t.Fatalf("poll interval = %s, want 1m", cfg.PollInterval)
	}
}

func TestParseAppConfigFlagsOverrideDefaults(t *testing.T) {
	cfg, err := parseAppConfig([]string{"--api-key", "flag-key", "--poll-interval", "15s", "--demo"}, func(string) string {
		return "env-key"
	})
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.APIKey != "flag-key" {
		t.Fatalf("API key = %q, want flag-key", cfg.APIKey)
	}
	if cfg.PollInterval != 15*time.Second {
		t.Fatalf("poll interval = %s, want 15s", cfg.PollInterval)
	}
	if !cfg.Demo {
		t.Fatal("demo flag should be true")
	}
}

func TestParseAppConfigRejectsInvalidInterval(t *testing.T) {
	_, err := parseAppConfig([]string{"--poll-interval", "0s"}, nil)
	if err == nil {
		t.Fatal("expected error for zero poll interval")
	}
}

func TestAppConfigProviderSelectsDemoWithoutKey(t *testing.T) {
	cfg := appConfig{}
	snapshot, err := cfg.provider().Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.Source != statusapi.SourceDemo {
		t.Fatalf("source = %s, want demo", snapshot.Source)
	}
}
