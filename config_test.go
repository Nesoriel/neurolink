package main

import (
	"context"
	"testing"
	"time"

	"neurolink/apex-server-monitor/statusapi"
	"neurolink/apex-server-monitor/tui"
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
	if cfg.Language != tui.LanguageEnglish {
		t.Fatalf("language = %s, want en", cfg.Language)
	}
}

func TestParseAppConfigFlagsOverrideDefaults(t *testing.T) {
	cfg, err := parseAppConfig([]string{"--api-key", "flag-key", "--poll-interval", "15s", "--demo"}, func(string) string {
		return ""
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
	if cfg.Language != tui.LanguageEnglish {
		t.Fatalf("language = %s, want en", cfg.Language)
	}
}

func TestParseAppConfigUsesEnvLanguage(t *testing.T) {
	cfg, err := parseAppConfig(nil, func(key string) string {
		if key == envLang {
			return "zh-Hans"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.Language != tui.LanguageSimplifiedChinese {
		t.Fatalf("language = %s, want zh-Hans", cfg.Language)
	}
}

func TestParseAppConfigLangFlagOverridesEnv(t *testing.T) {
	cfg, err := parseAppConfig([]string{"--lang", "en"}, func(key string) string {
		if key == envLang {
			return "zh-Hans"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.Language != tui.LanguageEnglish {
		t.Fatalf("language = %s, want en", cfg.Language)
	}
}

func TestParseAppConfigRejectsInvalidLanguage(t *testing.T) {
	_, err := parseAppConfig([]string{"--lang", "fr"}, nil)
	if err == nil {
		t.Fatal("expected error for invalid language")
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
