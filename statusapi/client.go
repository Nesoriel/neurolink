package statusapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.apexlegendsstatus.com"

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type ClientOptions struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(options ClientOptions) *Client {
	baseURL := strings.TrimRight(options.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		apiKey:     strings.TrimSpace(options.APIKey),
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) Fetch(ctx context.Context) (Snapshot, error) {
	if c.apiKey == "" {
		err := fmt.Errorf("status API key is not configured")
		return UnavailableSnapshot(SourceLive, err, time.Now()), err
	}

	endpoint, err := url.JoinPath(c.baseURL, "servers")
	if err != nil {
		return Snapshot{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Snapshot{}, err
	}
	request.Header.Set("Authorization", c.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "neurolink/0.1")

	response, err := c.httpClient.Do(request)
	if err != nil {
		snapshot := UnavailableSnapshot(SourceLive, err, time.Now())
		return snapshot, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err := fmt.Errorf("status API returned HTTP %d", response.StatusCode)
		snapshot := UnavailableSnapshot(SourceLive, err, time.Now())
		return snapshot, err
	}

	var raw map[string]any
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		snapshot := UnavailableSnapshot(SourceLive, err, time.Now())
		return snapshot, err
	}

	return NormalizeServerStatus(raw, SourceLive, time.Now()), nil
}
