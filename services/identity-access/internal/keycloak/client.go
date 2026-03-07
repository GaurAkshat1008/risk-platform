package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL string
	realm string
	clientID string
	clientSecret string

	logger *slog.Logger

	mu sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

type Config struct {
	BaseURL string
	Realm string
	ClientID string
	ClientSecret string
}

func NewClient(cfg Config, logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: cfg.BaseURL,
		realm: cfg.Realm,
		clientID: cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		logger: logger,
	}
}

func (c *Client) GetServiceToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.cachedToken != "" && time.Until(c.tokenExpiry) > 30*time.Second {
		token := c.cachedToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.fetchServiceToken(ctx)
}

func (c *Client) fetchServiceToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Until(c.tokenExpiry) > 30*time.Second {
		return c.cachedToken, nil
	}

	c.logger.Debug("Fetching new service account token")

	tokenURL := fmt.Sprintf(
		"%s/realms/%s/protocol/openid-connect/token",
		c.baseURL,
		c.realm,
	)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing token request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(body, &errResp)
		return "", fmt.Errorf(
			"keycloak token error [%d]: %s - %s",
			resp.StatusCode,
			errResp.Error,
			errResp.ErrorDescription,
		)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	c.cachedToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpireIn) * time.Second)
	c.logger.Debug("Obtained new service account token", "expires_in", tokenResp.ExpireIn)
	return c.cachedToken, nil
}

func (c *Client) doAdminRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	token, err := c.GetServiceToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting service token: %w", err)
	}
	fullUrl := fmt.Sprintf("%s/admin/realms/%s%s", c.baseURL, c.realm, path)

	req, err := http.NewRequestWithContext(ctx, method, fullUrl, body)
	if err != nil {
		return nil, fmt.Errorf("creating admin request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.GetServiceToken(ctx)
	return err
}