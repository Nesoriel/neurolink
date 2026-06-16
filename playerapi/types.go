package playerapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const Attribution = "Data from apexlegendsstatus.com"

type Provider interface {
	Lookup(ctx context.Context, request LookupRequest) (PlayerSnapshot, error)
}

type SourceMode string

const (
	SourceLive SourceMode = "LIVE"
	SourceDemo SourceMode = "DEMO"
)

type Platform string

const (
	PlatformPC  Platform = "PC"
	PlatformPS4 Platform = "PS4"
	PlatformX1  Platform = "X1"
)

var (
	ErrAPIKeyRequired  = errors.New("player lookup API key is not configured")
	ErrInvalidAPIKey   = errors.New("player lookup API key was rejected")
	ErrPlayerNotFound  = errors.New("player was not found")
	ErrUnknownPlatform = errors.New("unknown player platform")
	ErrRateLimited     = errors.New("player lookup rate limit reached")
	ErrExternalAPI     = errors.New("external player lookup API error")
	ErrInternalAPI     = errors.New("player lookup API internal error")
	ErrPlayerRequired  = errors.New("player name is required")
)

type LookupRequest struct {
	Player   string
	Platform Platform
}

type PlayerSnapshot struct {
	Source         SourceMode
	Attribution    string
	LookupAt       time.Time
	PlayerName     string
	Platform       Platform
	UID            string
	Level          int
	HasLevel       bool
	RankName       string
	RankDivision   string
	RankScore      int
	HasRank        bool
	SelectedLegend string
	Trackers       []Tracker
	Notice         string
}

type Tracker struct {
	Name  string
	Value string
}

func ParsePlatform(value string) (Platform, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case string(PlatformPC):
		return PlatformPC, nil
	case string(PlatformPS4), "PLAYSTATION", "PS5":
		return PlatformPS4, nil
	case string(PlatformX1), "XBOX":
		return PlatformX1, nil
	default:
		return "", fmt.Errorf("%w: %q (supported: PC, PS4, X1)", ErrUnknownPlatform, value)
	}
}

func SupportedPlatforms() []Platform {
	return []Platform{PlatformPC, PlatformPS4, PlatformX1}
}

func (r LookupRequest) normalized() (LookupRequest, error) {
	player := strings.TrimSpace(r.Player)
	if player == "" {
		return LookupRequest{}, ErrPlayerRequired
	}
	platform, err := ParsePlatform(string(r.Platform))
	if err != nil {
		return LookupRequest{}, err
	}
	return LookupRequest{Player: player, Platform: platform}, nil
}
