package statusapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  Status
	}{
		{input: "UP", want: StatusHealthy},
		{input: "RUNNING", want: StatusHealthy},
		{input: "Mostly operational", want: StatusHealthy},
		{input: "degraded performance", want: StatusDegraded},
		{input: "maintenance", want: StatusDegraded},
		{input: "DOWN", want: StatusDown},
		{input: "service outage", want: StatusDown},
		{input: "", want: StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeStatus(tt.input); got != tt.want {
				t.Fatalf("NormalizeStatus(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeServerStatusMapsCoreServices(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	raw := map[string]any{
		"ApexOauth_Crossplay": map[string]any{
			"EU-West": map[string]any{"Status": "UP", "HTTPCode": float64(200), "ResponseTime": float64(20), "QueryTimestamp": float64(now.Unix())},
		},
		"EA_novafusin": map[string]any{
			"US-East": map[string]any{"Status": "SLOW", "HTTPCode": float64(200), "ResponseTime": float64(180), "QueryTimestamp": float64(now.Unix())},
		},
		"Origin_login": map[string]any{
			"US-West": map[string]any{"Status": "UP", "HTTPCode": float64(200), "ResponseTime": float64(34)},
		},
		"EA_accounts": map[string]any{
			"EU-East": map[string]any{"Status": "DOWN", "HTTPCode": float64(503)},
		},
		"selfCoreTest": map[string]any{
			"Status-website": map[string]any{"Status": "UP", "HTTPCode": float64(200), "ResponseTime": float64(64)},
		},
		"lastReports": []any{
			map[string]any{"country": "US", "platform": "PC", "issue": "login"},
		},
	}

	snapshot := NormalizeServerStatus(raw, SourceLive, now)

	if snapshot.Source != SourceLive {
		t.Fatalf("source = %s, want %s", snapshot.Source, SourceLive)
	}
	if snapshot.Overall != StatusDown {
		t.Fatalf("overall = %s, want %s", snapshot.Overall, StatusDown)
	}
	if len(snapshot.Services) != len(CoreServices()) {
		t.Fatalf("services = %d, want %d", len(snapshot.Services), len(CoreServices()))
	}

	matchmaking, ok := ServiceByID(snapshot, ServiceMatchmaking)
	if !ok {
		t.Fatal("matchmaking service missing")
	}
	if matchmaking.Status != StatusDegraded {
		t.Fatalf("matchmaking status = %s, want %s", matchmaking.Status, StatusDegraded)
	}
	if len(matchmaking.Regions) != 1 || !matchmaking.Regions[0].HasLatency {
		t.Fatalf("matchmaking regions not normalized: %#v", matchmaking.Regions)
	}

	accounts, ok := ServiceByID(snapshot, ServicePlayerAccount)
	if !ok || accounts.Status != StatusDown {
		t.Fatalf("accounts = %#v, want down service", accounts)
	}
	if len(snapshot.RecentReports) != 1 || snapshot.RecentReports[0].Country != "US" {
		t.Fatalf("reports = %#v, want one US report", snapshot.RecentReports)
	}
}

func TestNormalizeServerStatusKeepsAPIHealthSeparateFromPlayableOverall(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	raw := map[string]any{
		"ApexOauth_Crossplay": map[string]any{"Asia": map[string]any{"Status": "UP", "HTTPCode": float64(200)}},
		"EA_novafusion":       map[string]any{"Asia": map[string]any{"Status": "UP", "HTTPCode": float64(200)}},
		"Origin_login":        map[string]any{"Asia": map[string]any{"Status": "UP", "HTTPCode": float64(200)}},
		"EA_accounts":         map[string]any{"Asia": map[string]any{"Status": "UP", "HTTPCode": float64(200)}},
		"selfCoreTest": map[string]any{
			"Status-website": map[string]any{"Status": "UP", "HTTPCode": float64(200)},
			"Overflow-#1":    map[string]any{"Status": "DOWN", "HTTPCode": float64(503)},
		},
	}

	snapshot := NormalizeServerStatus(raw, SourceLive, now)
	if snapshot.Overall != StatusHealthy {
		t.Fatalf("overall = %s, want healthy playable services despite API health issue", snapshot.Overall)
	}
	apiHealth, ok := ServiceByID(snapshot, ServiceAPIHealth)
	if !ok || apiHealth.Status != StatusDown {
		t.Fatalf("api health = %#v, want separate down API health card", apiHealth)
	}
}

func TestClientUsesAuthorizationHeaderAndNormalizesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers" {
			t.Fatalf("path = %s, want /servers", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Fatalf("Authorization header = %q, want test-key", got)
		}
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("Accept header = %q, want */* because upstream returns text/plain JSON", got)
		}
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		err := json.NewEncoder(w).Encode(map[string]any{
			"ApexOauth_Crossplay": map[string]any{"EU-West": map[string]any{"Status": "UP", "HTTPCode": 200}},
			"EA_novafusin":        map[string]any{"US-East": map[string]any{"Status": "UP", "HTTPCode": 200}},
			"Origin_login":        map[string]any{"US-West": map[string]any{"Status": "UP", "HTTPCode": 200}},
			"EA_accounts":         map[string]any{"EU-East": map[string]any{"Status": "UP", "HTTPCode": 200}},
			"selfCoreTest":        map[string]any{"Status-website": map[string]any{"Status": "UP", "HTTPCode": 200}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()})
	snapshot, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.Source != SourceLive {
		t.Fatalf("source = %s, want %s", snapshot.Source, SourceLive)
	}
	if snapshot.Overall != StatusHealthy {
		t.Fatalf("overall = %s, want %s", snapshot.Overall, StatusHealthy)
	}
}

func TestDemoProviderIsClearlyLabeled(t *testing.T) {
	snapshot, err := NewDemoProvider().Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.Source != SourceDemo {
		t.Fatalf("source = %s, want %s", snapshot.Source, SourceDemo)
	}
	if snapshot.Notice == "" {
		t.Fatal("demo snapshot should include an explicit notice")
	}
	if len(snapshot.Services) != len(CoreServices()) {
		t.Fatalf("services = %d, want %d", len(snapshot.Services), len(CoreServices()))
	}
}
