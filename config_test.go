package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nesoriel/neurolink/statusapi"
	"github.com/Nesoriel/neurolink/tui"
)

func TestParseAppConfigUsesEnvAPIKey(t *testing.T) {
	cfg, err := parseAppConfig(withConfigPath(t, nil), testLookupEnv(map[string]string{
		envAPIKey: "env-key",
	}))
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

func TestParseAppConfigLoadsConfigFile(t *testing.T) {
	path := testConfigPath(t)
	if err := saveFileConfig(path, fileConfig{
		APIKey:       "file-key",
		Language:     "zh-Hans",
		PollInterval: "45s",
	}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	cfg, err := parseAppConfig([]string{"--config", path}, nil)
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.APIKey != "file-key" {
		t.Fatalf("API key = %q, want file-key", cfg.APIKey)
	}
	if cfg.PollInterval != 45*time.Second {
		t.Fatalf("poll interval = %s, want 45s", cfg.PollInterval)
	}
	if cfg.Language != tui.LanguageSimplifiedChinese {
		t.Fatalf("language = %s, want zh-Hans", cfg.Language)
	}
	if cfg.ConfigPath != filepath.Clean(path) {
		t.Fatalf("config path = %q, want %q", cfg.ConfigPath, filepath.Clean(path))
	}
}

func TestParseAppConfigEnvOverridesConfigFile(t *testing.T) {
	path := testConfigPath(t)
	if err := saveFileConfig(path, fileConfig{
		APIKey:       "file-key",
		Language:     "zh-Hans",
		PollInterval: "45s",
	}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	cfg, err := parseAppConfig([]string{"--config", path}, testLookupEnv(map[string]string{
		envAPIKey:       "env-key",
		envLang:         "en",
		envPollInterval: "20s",
	}))
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("API key = %q, want env-key", cfg.APIKey)
	}
	if cfg.PollInterval != 20*time.Second {
		t.Fatalf("poll interval = %s, want 20s", cfg.PollInterval)
	}
	if cfg.Language != tui.LanguageEnglish {
		t.Fatalf("language = %s, want en", cfg.Language)
	}
}

func TestParseAppConfigFlagsOverrideEnvAndConfig(t *testing.T) {
	path := testConfigPath(t)
	if err := saveFileConfig(path, fileConfig{
		APIKey:       "file-key",
		Language:     "zh-Hans",
		PollInterval: "45s",
	}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	cfg, err := parseAppConfig([]string{
		"--config", path,
		"--api-key", "flag-key",
		"--poll-interval", "15s",
		"--lang", "en",
		"--demo",
	}, testLookupEnv(map[string]string{
		envAPIKey:       "env-key",
		envLang:         "zh-Hans",
		envPollInterval: "20s",
	}))
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.APIKey != "flag-key" {
		t.Fatalf("API key = %q, want flag-key", cfg.APIKey)
	}
	if cfg.PollInterval != 15*time.Second {
		t.Fatalf("poll interval = %s, want 15s", cfg.PollInterval)
	}
	if cfg.Language != tui.LanguageEnglish {
		t.Fatalf("language = %s, want en", cfg.Language)
	}
	if !cfg.Demo {
		t.Fatal("demo flag should be true")
	}
}

func TestConfigPathCanComeFromEnv(t *testing.T) {
	path := testConfigPath(t)
	if err := saveFileConfig(path, fileConfig{Language: "zh-Hans"}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	cfg, err := parseAppConfig(nil, testLookupEnv(map[string]string{
		envConfigPath: path,
	}))
	if err != nil {
		t.Fatalf("parseAppConfig() error = %v", err)
	}
	if cfg.ConfigPath != filepath.Clean(path) {
		t.Fatalf("config path = %q, want %q", cfg.ConfigPath, filepath.Clean(path))
	}
	if cfg.Language != tui.LanguageSimplifiedChinese {
		t.Fatalf("language = %s, want zh-Hans", cfg.Language)
	}
}

func TestSaveLoadConfigAndMasking(t *testing.T) {
	path := testConfigPath(t)
	secret := "not-a-real-key-1234"
	if err := saveFileConfig(path, fileConfig{
		APIKey:       "  " + secret + "  ",
		Language:     "zh-Hans",
		PollInterval: "30s",
	}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	loaded, err := loadFileConfig(path)
	if err != nil {
		t.Fatalf("loadFileConfig() error = %v", err)
	}
	if loaded.APIKey != secret {
		t.Fatalf("API key = %q, want trimmed secret", loaded.APIKey)
	}
	if loaded.Language != "zh-Hans" {
		t.Fatalf("language = %q, want zh-Hans", loaded.Language)
	}
	if loaded.PollInterval != "30s" {
		t.Fatalf("poll interval = %q, want 30s", loaded.PollInterval)
	}

	var out bytes.Buffer
	if err := printConfig(&out, path, loaded); err != nil {
		t.Fatalf("printConfig() error = %v", err)
	}
	output := out.String()
	if strings.Contains(output, secret) {
		t.Fatalf("config output leaked API key: %q", output)
	}
	if !strings.Contains(output, "********1234") {
		t.Fatalf("config output = %q, want masked key suffix", output)
	}
}

func TestConfigSetShowAndUnsetCommandsMaskAPIKey(t *testing.T) {
	path := testConfigPath(t)
	secret := "not-a-real-key-1234"

	var setOut bytes.Buffer
	err := runConfigCommand([]string{
		"config", "set",
		"--config", path,
		"--api-key", secret,
		"--lang", "zh-Hans",
		"--poll-interval", "25s",
	}, nil, &setOut)
	if err != nil {
		t.Fatalf("runConfigCommand(set) error = %v", err)
	}
	if strings.Contains(setOut.String(), secret) {
		t.Fatalf("set output leaked API key: %q", setOut.String())
	}

	loaded, err := loadFileConfig(path)
	if err != nil {
		t.Fatalf("loadFileConfig() error = %v", err)
	}
	if loaded.APIKey != secret || loaded.Language != "zh-Hans" || loaded.PollInterval != "25s" {
		t.Fatalf("loaded config = %#v, want persisted values", loaded)
	}

	var showOut bytes.Buffer
	if err := runConfigCommand([]string{"config", "show", "--config", path}, nil, &showOut); err != nil {
		t.Fatalf("runConfigCommand(show) error = %v", err)
	}
	if strings.Contains(showOut.String(), secret) {
		t.Fatalf("show output leaked API key: %q", showOut.String())
	}
	if !strings.Contains(showOut.String(), "********1234") {
		t.Fatalf("show output = %q, want masked key", showOut.String())
	}

	var unsetOut bytes.Buffer
	if err := runConfigCommand([]string{"config", "unset", "--config", path, "api-key"}, nil, &unsetOut); err != nil {
		t.Fatalf("runConfigCommand(unset) error = %v", err)
	}
	loaded, err = loadFileConfig(path)
	if err != nil {
		t.Fatalf("loadFileConfig() error = %v", err)
	}
	if loaded.APIKey != "" {
		t.Fatalf("API key = %q, want unset", loaded.APIKey)
	}
	if !strings.Contains(unsetOut.String(), "api-key: (unset)") {
		t.Fatalf("unset output = %q, want unset key", unsetOut.String())
	}
}

func TestConfigSetKeyValueCommand(t *testing.T) {
	path := testConfigPath(t)
	var out bytes.Buffer
	if err := runConfigCommand([]string{"config", "set", "--config", path, "poll-interval", "40s"}, nil, &out); err != nil {
		t.Fatalf("runConfigCommand(set key value) error = %v", err)
	}

	loaded, err := loadFileConfig(path)
	if err != nil {
		t.Fatalf("loadFileConfig() error = %v", err)
	}
	if loaded.PollInterval != "40s" {
		t.Fatalf("poll interval = %q, want 40s", loaded.PollInterval)
	}
}

func TestParseAppConfigRejectsInvalidLanguage(t *testing.T) {
	_, err := parseAppConfig(withConfigPath(t, []string{"--lang", "fr"}), nil)
	if err == nil {
		t.Fatal("expected error for invalid language")
	}
}

func TestParseAppConfigRejectsInvalidInterval(t *testing.T) {
	_, err := parseAppConfig(withConfigPath(t, []string{"--poll-interval", "0s"}), nil)
	if err == nil {
		t.Fatal("expected error for zero poll interval")
	}
}

func TestParseAppConfigRejectsInvalidConfigInterval(t *testing.T) {
	path := testConfigPath(t)
	if err := saveFileConfig(path, fileConfig{PollInterval: "0s"}); err != nil {
		t.Fatalf("saveFileConfig() error = %v", err)
	}

	_, err := parseAppConfig([]string{"--config", path}, nil)
	if err == nil {
		t.Fatal("expected error for zero config poll interval")
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

func withConfigPath(t *testing.T, args []string) []string {
	t.Helper()
	result := []string{"--config", testConfigPath(t)}
	return append(result, args...)
}

func testConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "neurolink", "config.json")
}

func testLookupEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
