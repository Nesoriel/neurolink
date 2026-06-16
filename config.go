package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"neurolink/apex-server-monitor/statusapi"
	"neurolink/apex-server-monitor/tui"
)

const envAPIKey = "NEUROLINK_APEX_API_KEY"
const envLang = "NEUROLINK_LANG"

type appConfig struct {
	APIKey       string
	PollInterval time.Duration
	Demo         bool
	Language     tui.Language
}

func parseAppConfig(args []string, lookupEnv func(string) string) (appConfig, error) {
	if lookupEnv == nil {
		lookupEnv = func(string) string { return "" }
	}

	langValue := strings.TrimSpace(lookupEnv(envLang))
	if langValue == "" {
		langValue = string(tui.LanguageEnglish)
	}

	cfg := appConfig{
		APIKey:       strings.TrimSpace(lookupEnv(envAPIKey)),
		PollInterval: time.Minute,
	}

	flags := flag.NewFlagSet("neurolink", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "Apex Legends Status API key")
	flags.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "service status poll interval")
	flags.BoolVar(&cfg.Demo, "demo", false, "use deterministic demo data instead of live API data")
	flags.StringVar(&langValue, "lang", langValue, "UI language: en or zh-Hans")

	if err := flags.Parse(args); err != nil {
		return appConfig{}, err
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

func (c appConfig) provider() statusapi.Provider {
	if c.Demo || c.APIKey == "" {
		return statusapi.NewDemoProvider()
	}
	return statusapi.NewClient(statusapi.ClientOptions{APIKey: c.APIKey})
}
