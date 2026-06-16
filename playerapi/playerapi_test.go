package playerapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientUsesBridgeRequestAndAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bridge" {
			t.Fatalf("path = %s, want /bridge", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Fatalf("Authorization header = %q, want test-key", got)
		}
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("Accept header = %q, want */*", got)
		}
		query := r.URL.Query()
		if got := query.Get("player"); got != "Some Origin Name" {
			t.Fatalf("player query = %q, want player name", got)
		}
		if got := query.Get("platform"); got != "PC" {
			t.Fatalf("platform query = %q, want PC", got)
		}
		if got := query.Get("version"); got != "5" {
			t.Fatalf("version query = %q, want 5", got)
		}

		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		err := json.NewEncoder(w).Encode(bridgePayload())
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()})
	snapshot, err := client.Lookup(context.Background(), LookupRequest{Player: " Some Origin Name ", Platform: PlatformPC})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if snapshot.Source != SourceLive {
		t.Fatalf("source = %s, want %s", snapshot.Source, SourceLive)
	}
	if snapshot.PlayerName != "Some Origin Name" || snapshot.UID != "1000000000001" {
		t.Fatalf("identity = %#v, want normalized player", snapshot)
	}
}

func TestNormalizeBridgeResponseExtractsPlayerSummary(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	snapshot := NormalizeBridgeResponse(bridgePayload(), LookupRequest{Player: "fallback", Platform: PlatformPC}, SourceLive, now)

	if snapshot.PlayerName != "Some Origin Name" {
		t.Fatalf("player name = %q, want Some Origin Name", snapshot.PlayerName)
	}
	if snapshot.Platform != PlatformPC {
		t.Fatalf("platform = %s, want PC", snapshot.Platform)
	}
	if !snapshot.HasLevel || snapshot.Level != 512 {
		t.Fatalf("level = %d has=%t, want level 512", snapshot.Level, snapshot.HasLevel)
	}
	if !snapshot.HasRank || snapshot.RankName != "Platinum" || snapshot.RankDivision != "3" || snapshot.RankScore != 9280 {
		t.Fatalf("rank = %#v, want Platinum 3 / 9280", snapshot)
	}
	if snapshot.SelectedLegend != "Crypto" {
		t.Fatalf("selected legend = %q, want Crypto", snapshot.SelectedLegend)
	}
	if len(snapshot.Trackers) != 3 {
		t.Fatalf("trackers = %#v, want 3 normalized trackers", snapshot.Trackers)
	}
	if snapshot.LookupAt != now {
		t.Fatalf("lookup time = %s, want %s", snapshot.LookupAt, now)
	}
}

func TestClientMapsDocumentedErrors(t *testing.T) {
	tests := []struct {
		name string
		code int
		want error
	}{
		{name: "invalid key", code: http.StatusForbidden, want: ErrInvalidAPIKey},
		{name: "not found", code: http.StatusNotFound, want: ErrPlayerNotFound},
		{name: "unknown platform", code: http.StatusGone, want: ErrUnknownPlatform},
		{name: "rate limit", code: http.StatusTooManyRequests, want: ErrRateLimited},
		{name: "external api", code: http.StatusMethodNotAllowed, want: ErrExternalAPI},
		{name: "internal api", code: http.StatusInternalServerError, want: ErrInternalAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "problem", tt.code)
			}))
			defer server.Close()

			client := NewClient(ClientOptions{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()})
			_, err := client.Lookup(context.Background(), LookupRequest{Player: "somebody", Platform: PlatformPC})
			if !errors.Is(err, tt.want) {
				t.Fatalf("Lookup() error = %v, want errors.Is %v", err, tt.want)
			}
		})
	}
}

func TestClientRejectsMissingKeyAndUnknownPlatformBeforeRequest(t *testing.T) {
	client := NewClient(ClientOptions{})
	_, err := client.Lookup(context.Background(), LookupRequest{Player: "somebody", Platform: PlatformPC})
	if !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("missing key error = %v, want ErrAPIKeyRequired", err)
	}

	_, err = client.Lookup(context.Background(), LookupRequest{Player: "somebody", Platform: Platform("SWITCH")})
	if !errors.Is(err, ErrUnknownPlatform) {
		t.Fatalf("unknown platform error = %v, want ErrUnknownPlatform", err)
	}
}

func TestClientMapsPayloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(map[string]any{"Error": "The player could not be found."})
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{APIKey: "test-key", BaseURL: server.URL, HTTPClient: server.Client()})
	_, err := client.Lookup(context.Background(), LookupRequest{Player: "nobody", Platform: PlatformPS4})
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("payload error = %v, want ErrPlayerNotFound", err)
	}
}

func TestDemoProviderIsExplicitlyLabeled(t *testing.T) {
	snapshot, err := NewDemoProvider().Lookup(context.Background(), LookupRequest{Player: "Demo", Platform: PlatformX1})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if snapshot.Source != SourceDemo {
		t.Fatalf("source = %s, want DEMO", snapshot.Source)
	}
	if snapshot.Notice == "" {
		t.Fatal("demo lookup should include a clear notice")
	}
	if snapshot.PlayerName != "Demo" || snapshot.Platform != PlatformX1 {
		t.Fatalf("snapshot = %#v, want requested player/platform", snapshot)
	}
}

func bridgePayload() map[string]any {
	return map[string]any{
		"global": map[string]any{
			"name":     "Some Origin Name",
			"uid":      "1000000000001",
			"platform": "PC",
			"level":    float64(512),
			"rank": map[string]any{
				"rankName":  "Platinum",
				"rankDiv":   float64(3),
				"rankScore": float64(9280),
			},
		},
		"legends": map[string]any{
			"selected": map[string]any{
				"LegendName": "Crypto",
				"data": []any{
					map[string]any{"name": "Wins", "value": float64(42)},
				},
			},
		},
		"total": map[string]any{
			"kills":  map[string]any{"name": "Kills", "value": float64(1234)},
			"damage": map[string]any{"name": "Damage", "value": float64(456789)},
		},
	}
}
