package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	AuthURL  = "https://auth.openai.com/oauth/authorize"
	TokenURL = "https://auth.openai.com/oauth/token"
	ClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	UsageURL = "https://chatgpt.com/backend-api/wham/usage"
	ua       = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
)

var (
	ErrTokenExpired = errors.New("token_expired")
)

type Client struct {
	Authorization string `json:"authorization"`
	RefreshToken  string `json:"refresh_token"`
	AccountId     string `json:"account_id"`
	UserAgent     string
	Client        *http.Client
}

func NewClient(authorization, refreshToken, accountId string, httpClient *http.Client) *Client {
	return &Client{
		Authorization: authorization,
		RefreshToken:  refreshToken,
		AccountId:     accountId,
		UserAgent:     ua,
		Client:        httpClient,
	}
}

func (c *Client) GetUsage(ctx context.Context) (*UsageResponse, error) {
	var userAgent string
	if c.UserAgent == "" {
		userAgent = ua
	} else {
		userAgent = c.UserAgent
	}
	headers := map[string]string{
		"Authorization":      c.Authorization,
		"Chatgpt-Account-Id": c.AccountId,
		"Content-Type":       "application/json",
		"User-Agent":         userAgent,
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, UsageURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	response, err := c.Client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read usage response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("token usage failed with status %d: %s", response.StatusCode, string(body))
	}
	var usageResponse UsageResponse
	err = json.Unmarshal(body, &usageResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse usage response: %w", err)
	}
	return &usageResponse, nil
}

func (c *Client) RefreshTokens(ctx context.Context) (*TokenData, error) {

	data := url.Values{
		"client_id":     {ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.RefreshToken},
		"scope":         {"openid profile email"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Extract account ID from ID token
	claims, err := ParseJWTToken(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refreshed ID token: %w", err)
	}

	accountID := ""
	email := ""
	if claims != nil {
		accountID = claims.GetAccountID()
		email = claims.Email
	}

	return &TokenData{
		IDToken:      tokenResp.IDToken,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		AccountID:    accountID,
		Email:        email,
		Expire:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}
