package playerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func (c *Client) Lookup(ctx context.Context, request LookupRequest) (PlayerSnapshot, error) {
	request, err := request.normalized()
	if err != nil {
		return PlayerSnapshot{}, err
	}
	if c.apiKey == "" {
		return PlayerSnapshot{}, ErrAPIKeyRequired
	}

	endpoint, err := url.JoinPath(c.baseURL, "bridge")
	if err != nil {
		return PlayerSnapshot{}, err
	}
	values := url.Values{}
	values.Set("player", request.Player)
	values.Set("platform", string(request.Platform))
	values.Set("version", "5")

	requestURL := endpoint + "?" + values.Encode()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return PlayerSnapshot{}, err
	}
	httpRequest.Header.Set("Authorization", c.apiKey)
	httpRequest.Header.Set("Accept", "*/*")
	httpRequest.Header.Set("User-Agent", "neurolink/0.1")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return PlayerSnapshot{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, response.Body)
		return PlayerSnapshot{}, errorForStatus(response.StatusCode)
	}

	var raw map[string]any
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return PlayerSnapshot{}, fmt.Errorf("decode player lookup response: %w", err)
	}
	if err := errorFromPayload(raw); err != nil {
		return PlayerSnapshot{}, err
	}
	return NormalizeBridgeResponse(raw, request, SourceLive, time.Now()), nil
}

func errorForStatus(code int) error {
	switch code {
	case http.StatusForbidden:
		return fmt.Errorf("%w: HTTP %d", ErrInvalidAPIKey, code)
	case http.StatusNotFound:
		return fmt.Errorf("%w: HTTP %d", ErrPlayerNotFound, code)
	case http.StatusGone:
		return fmt.Errorf("%w: HTTP %d", ErrUnknownPlatform, code)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: HTTP %d", ErrRateLimited, code)
	case http.StatusBadRequest:
		return fmt.Errorf("%w: HTTP %d", ErrExternalAPI, code)
	case http.StatusMethodNotAllowed:
		return fmt.Errorf("%w: HTTP %d", ErrExternalAPI, code)
	case http.StatusInternalServerError:
		return fmt.Errorf("%w: HTTP %d", ErrInternalAPI, code)
	default:
		if code >= 500 {
			return fmt.Errorf("%w: HTTP %d", ErrInternalAPI, code)
		}
		return fmt.Errorf("%w: HTTP %d", ErrExternalAPI, code)
	}
}

func errorFromPayload(raw map[string]any) error {
	message := strings.ToLower(firstString(raw, "Error", "error", "message", "Message"))
	if message == "" {
		return nil
	}

	switch {
	case strings.Contains(message, "unknown api"),
		strings.Contains(message, "unauthorized"),
		strings.Contains(message, "invalid api"):
		return ErrInvalidAPIKey
	case strings.Contains(message, "not found"),
		strings.Contains(message, "could not be found"):
		return ErrPlayerNotFound
	case strings.Contains(message, "unknown platform"):
		return ErrUnknownPlatform
	case strings.Contains(message, "rate"):
		return ErrRateLimited
	default:
		return fmt.Errorf("%w: %s", ErrExternalAPI, message)
	}
}
