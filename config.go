package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nesoriel/neurolink/playerapi"
	"github.com/Nesoriel/neurolink/statusapi"
	"github.com/Nesoriel/neurolink/tui"
)

const (
	envAPIKey       = "NEUROLINK_APEX_API_KEY"
	envConfigPath   = "NEUROLINK_CONFIG"
	envLang         = "NEUROLINK_LANG"
	envPollInterval = "NEUROLINK_POLL_INTERVAL"

	configDirName  = "neurolink"
	configFileName = "config.json"
)

const defaultPollInterval = time.Minute

type appConfig struct {
	APIKey       string
	PollInterval time.Duration
	Demo         bool
	Language     tui.Language
	ConfigPath   string
}

type fileConfig struct {
	APIKey       string `json:"api_key,omitempty"`
	Language     string `json:"language,omitempty"`
	PollInterval string `json:"poll_interval,omitempty"`
}

func parseAppConfig(args []string, lookupEnv func(string) string) (appConfig, error) {
	lookupEnv = normalizeLookupEnv(lookupEnv)

	configPath, err := configPathFromArgs(args, lookupEnv)
	if err != nil {
		return appConfig{}, err
	}

	saved, err := loadFileConfig(configPath)
	if err != nil {
		return appConfig{}, err
	}

	cfg := defaultAppConfig(configPath)
	if err := applyFileConfig(&cfg, saved); err != nil {
		return appConfig{}, err
	}
	if err := applyEnvConfig(&cfg, lookupEnv); err != nil {
		return appConfig{}, err
	}

	langValue := string(cfg.Language)
	flags := flag.NewFlagSet("neurolink", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "configuration file path")
	flags.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "Apex Legends Status API key")
	flags.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "service status poll interval")
	flags.BoolVar(&cfg.Demo, "demo", false, "use deterministic demo data instead of live API data")
	flags.StringVar(&langValue, "lang", langValue, "UI language: en or zh-Hans")

	if err := flags.Parse(args); err != nil {
		return appConfig{}, err
	}
	if flags.NArg() > 0 {
		return appConfig{}, fmt.Errorf("unknown command or argument %q", flags.Arg(0))
	}

	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	language, err := tui.ParseLanguage(langValue)
	if err != nil {
		return appConfig{}, err
	}
	cfg.Language = language
	if cfg.PollInterval <= 0 {
		return appConfig{}, fmt.Errorf("poll interval must be greater than zero")
	}
	return cfg, nil
}

func defaultAppConfig(configPath string) appConfig {
	return appConfig{
		PollInterval: defaultPollInterval,
		Language:     tui.LanguageEnglish,
		ConfigPath:   configPath,
	}
}

func (c appConfig) provider() statusapi.Provider {
	if c.Demo || c.APIKey == "" {
		return statusapi.NewDemoProvider()
	}
	return statusapi.NewClient(statusapi.ClientOptions{APIKey: c.APIKey})
}

func (c appConfig) playerProvider() playerapi.Provider {
	if c.Demo || c.APIKey == "" {
		return playerapi.NewDemoProvider()
	}
	return playerapi.NewClient(playerapi.ClientOptions{APIKey: c.APIKey})
}

func defaultConfigPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(root, configDirName, configFileName), nil
}

func configPathFromArgs(args []string, lookupEnv func(string) string) (string, error) {
	lookupEnv = normalizeLookupEnv(lookupEnv)

	path := strings.TrimSpace(lookupEnv(envConfigPath))
	if path == "" {
		defaultPath, err := defaultConfigPath()
		if err != nil {
			return "", err
		}
		path = defaultPath
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return "", fmt.Errorf("--config requires a path")
			}
			path = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(arg, "--config="):
			path = strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
		}
		if path == "" {
			return "", fmt.Errorf("--config requires a path")
		}
	}

	return filepath.Clean(path), nil
}

func loadFileConfig(path string) (fileConfig, error) {
	var cfg fileConfig
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return cfg, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config %s: %w", path, err)
	}
	return normalizeFileConfig(cfg), nil
}

func saveFileConfig(path string, cfg fileConfig) error {
	cfg = normalizeFileConfig(cfg)
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	body = append(body, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure config %s: %w", path, err)
	}
	return nil
}

func normalizeFileConfig(cfg fileConfig) fileConfig {
	return fileConfig{
		APIKey:       strings.TrimSpace(cfg.APIKey),
		Language:     strings.TrimSpace(cfg.Language),
		PollInterval: strings.TrimSpace(cfg.PollInterval),
	}
}

func applyFileConfig(cfg *appConfig, saved fileConfig) error {
	saved = normalizeFileConfig(saved)
	if saved.APIKey != "" {
		cfg.APIKey = saved.APIKey
	}
	if saved.Language != "" {
		language, err := tui.ParseLanguage(saved.Language)
		if err != nil {
			return fmt.Errorf("config language: %w", err)
		}
		cfg.Language = language
	}
	if saved.PollInterval != "" {
		interval, err := parsePositiveDuration(saved.PollInterval, "config poll interval")
		if err != nil {
			return err
		}
		cfg.PollInterval = interval
	}
	return nil
}

func applyEnvConfig(cfg *appConfig, lookupEnv func(string) string) error {
	if value := strings.TrimSpace(lookupEnv(envAPIKey)); value != "" {
		cfg.APIKey = value
	}
	if value := strings.TrimSpace(lookupEnv(envLang)); value != "" {
		language, err := tui.ParseLanguage(value)
		if err != nil {
			return fmt.Errorf("%s: %w", envLang, err)
		}
		cfg.Language = language
	}
	if value := strings.TrimSpace(lookupEnv(envPollInterval)); value != "" {
		interval, err := parsePositiveDuration(value, envPollInterval)
		if err != nil {
			return err
		}
		cfg.PollInterval = interval
	}
	return nil
}

func parsePositiveDuration(value string, label string) (time.Duration, error) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", label, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", label)
	}
	return duration, nil
}

func isConfigCommand(args []string) bool {
	return configCommandIndex(args) >= 0
}

func configCommandIndex(args []string) int {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return -1
		}
		if arg == "config" {
			return i
		}
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		if cut, _, ok := strings.Cut(name, "="); ok {
			name = cut
		}
		switch name {
		case "api-key", "config", "lang", "poll-interval":
			if !strings.Contains(arg, "=") {
				i++
			}
		}
	}
	return -1
}

func runConfigCommand(args []string, lookupEnv func(string) string, stdout io.Writer) error {
	lookupEnv = normalizeLookupEnv(lookupEnv)
	index := configCommandIndex(args)
	if index < 0 {
		return fmt.Errorf("missing config command")
	}

	configPath, err := configPathFromArgs(args, lookupEnv)
	if err != nil {
		return err
	}

	subArgs := args[index+1:]
	if len(subArgs) == 0 || subArgs[0] == "help" || subArgs[0] == "--help" || subArgs[0] == "-h" {
		printConfigUsage(stdout)
		return nil
	}

	switch subArgs[0] {
	case "show":
		return runConfigShow(configPath, subArgs[1:], stdout)
	case "path":
		return runConfigPath(configPath, subArgs[1:], stdout)
	case "set":
		return runConfigSet(configPath, subArgs[1:], stdout)
	case "unset":
		return runConfigUnset(configPath, subArgs[1:], stdout)
	default:
		return fmt.Errorf("unknown config command %q", subArgs[0])
	}
}

func runConfigShow(configPath string, args []string, stdout io.Writer) error {
	flags := newConfigSubcommandFlagSet("neurolink config show", configPath)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	saved, err := loadFileConfig(configPath)
	if err != nil {
		return err
	}
	return printConfig(stdout, configPath, saved)
}

func runConfigPath(configPath string, args []string, stdout io.Writer) error {
	flags := newConfigSubcommandFlagSet("neurolink config path", configPath)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	_, err := fmt.Fprintln(stdout, configPath)
	return err
}

func runConfigSet(configPath string, args []string, stdout io.Writer) error {
	args, err := stripConfigFlagArgs(args)
	if err != nil {
		return err
	}

	saved, err := loadFileConfig(configPath)
	if err != nil {
		return err
	}

	var changed bool
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if len(args) != 2 {
			return fmt.Errorf("usage: neurolink config set <api-key|language|poll-interval> <value>")
		}
		if _, err := setFileConfigValue(&saved, args[0], args[1]); err != nil {
			return err
		}
		changed = true
	} else {
		flags := newConfigSetFlagSet(configPath)
		values := configSetFlagValues{}
		registerConfigSetFlags(flags, &values)
		if err := flags.Parse(args); err != nil {
			return err
		}
		if flags.NArg() > 0 {
			return fmt.Errorf("unexpected argument %q", flags.Arg(0))
		}

		explicit := map[string]bool{}
		flags.Visit(func(f *flag.Flag) {
			explicit[f.Name] = true
		})
		if explicit["api-key"] {
			if _, err := setFileConfigValue(&saved, "api-key", values.APIKey); err != nil {
				return err
			}
			changed = true
		}
		if explicit["lang"] {
			if _, err := setFileConfigValue(&saved, "language", values.Language); err != nil {
				return err
			}
			changed = true
		}
		if explicit["poll-interval"] {
			if _, err := setFileConfigValue(&saved, "poll-interval", values.PollInterval); err != nil {
				return err
			}
			changed = true
		}
	}

	if !changed {
		return fmt.Errorf("nothing to save; set api-key, language, or poll-interval")
	}
	if err := saveFileConfig(configPath, saved); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Saved config to %s\n", configPath)
	return printConfig(stdout, configPath, saved)
}

func runConfigUnset(configPath string, args []string, stdout io.Writer) error {
	flags := newConfigSubcommandFlagSet("neurolink config unset", configPath)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() == 0 {
		return fmt.Errorf("usage: neurolink config unset <api-key|language|poll-interval|all> [...]")
	}

	saved, err := loadFileConfig(configPath)
	if err != nil {
		return err
	}
	for _, key := range flags.Args() {
		if err := unsetFileConfigValue(&saved, key); err != nil {
			return err
		}
	}
	if err := saveFileConfig(configPath, saved); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Updated config at %s\n", configPath)
	return printConfig(stdout, configPath, saved)
}

type configSetFlagValues struct {
	APIKey       string
	Language     string
	PollInterval string
}

func newConfigSubcommandFlagSet(name string, configPath string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.String("config", configPath, "configuration file path")
	return flags
}

func newConfigSetFlagSet(configPath string) *flag.FlagSet {
	return newConfigSubcommandFlagSet("neurolink config set", configPath)
}

func registerConfigSetFlags(flags *flag.FlagSet, values *configSetFlagValues) {
	flags.StringVar(&values.APIKey, "api-key", "", "persist Apex Legends Status API key")
	flags.StringVar(&values.Language, "lang", "", "persist UI language: en or zh-Hans")
	flags.StringVar(&values.PollInterval, "poll-interval", "", "persist service status poll interval")
}

func stripConfigFlagArgs(args []string) ([]string, error) {
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--config requires a path")
			}
			i++
		case strings.HasPrefix(arg, "--config="):
			continue
		default:
			result = append(result, arg)
		}
	}
	return result, nil
}

func setFileConfigValue(cfg *fileConfig, key string, value string) (string, error) {
	value = strings.TrimSpace(value)
	switch normalizeConfigKey(key) {
	case "api-key":
		if value == "" {
			return "", fmt.Errorf("api-key cannot be empty")
		}
		cfg.APIKey = value
		return "api-key", nil
	case "language":
		language, err := tui.ParseLanguage(value)
		if err != nil {
			return "", err
		}
		cfg.Language = string(language)
		return "language", nil
	case "poll-interval":
		interval, err := parsePositiveDuration(value, "poll interval")
		if err != nil {
			return "", err
		}
		cfg.PollInterval = interval.String()
		return "poll-interval", nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func unsetFileConfigValue(cfg *fileConfig, key string) error {
	switch normalizeConfigKey(key) {
	case "api-key":
		cfg.APIKey = ""
	case "language":
		cfg.Language = ""
	case "poll-interval":
		cfg.PollInterval = ""
	case "all":
		*cfg = fileConfig{}
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

func normalizeConfigKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "api-key", "apikey", "api_key", "key":
		return "api-key"
	case "lang", "language":
		return "language"
	case "poll-interval", "poll_interval", "interval":
		return "poll-interval"
	case "all":
		return "all"
	default:
		return strings.TrimSpace(key)
	}
}

func printConfig(stdout io.Writer, configPath string, saved fileConfig) error {
	saved = normalizeFileConfig(saved)

	language := saved.Language
	if language == "" {
		language = string(tui.LanguageEnglish) + " (default)"
	} else if parsed, err := tui.ParseLanguage(language); err != nil {
		return fmt.Errorf("config language: %w", err)
	} else {
		language = string(parsed)
	}

	pollInterval := saved.PollInterval
	if pollInterval == "" {
		pollInterval = defaultPollInterval.String() + " (default)"
	} else if parsed, err := parsePositiveDuration(pollInterval, "config poll interval"); err != nil {
		return err
	} else {
		pollInterval = parsed.String()
	}

	_, err := fmt.Fprintf(stdout, "Config file: %s\napi-key: %s\nlanguage: %s\npoll-interval: %s\n", configPath, maskAPIKey(saved.APIKey), language, pollInterval)
	return err
}

func maskAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(unset)"
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value)-2) + value[len(value)-2:]
	}
	return "********" + value[len(value)-4:]
}

func printConfigUsage(stdout io.Writer) {
	fmt.Fprintln(stdout, `Usage:
  neurolink [--api-key KEY] [--poll-interval 1m] [--lang en|zh-Hans] [--demo]
  neurolink config show [--config PATH]
  neurolink config path [--config PATH]
  neurolink config set api-key KEY
  neurolink config set language en|zh-Hans
  neurolink config set poll-interval 30s
  neurolink config set --api-key KEY --lang zh-Hans --poll-interval 30s
  neurolink config unset api-key|language|poll-interval|all

Normal run precedence: defaults < config file < environment < flags.
Environment: NEUROLINK_APEX_API_KEY, NEUROLINK_LANG, NEUROLINK_POLL_INTERVAL, NEUROLINK_CONFIG.`)
}

func normalizeLookupEnv(lookupEnv func(string) string) func(string) string {
	if lookupEnv == nil {
		return func(string) string { return "" }
	}
	return lookupEnv
}
