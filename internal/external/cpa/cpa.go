package cpa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	UploadModeJSON      = "json"
	UploadModeMultipart = "multipart"
	defaultTimeout      = 60 * time.Second
)

var (
	ErrNotFound = errors.New("not found")
)

type Client struct {
	baseURL       string
	managementKey string
	uploadMode    string
	httpClient    *http.Client
}

func NewCpaClient(baseURL, managementKey, uploadMode string, httpClient *http.Client) *Client {
	client := &Client{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		managementKey: strings.TrimSpace(managementKey),
		uploadMode:    strings.ToLower(strings.TrimSpace(uploadMode)),
		httpClient:    httpClient,
	}
	if client.uploadMode == "" {
		client.uploadMode = UploadModeJSON
	}
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return client
}

func (c *Client) UploadAuthFile(ctx context.Context, name string, content []byte) error {
	if c == nil {
		return fmt.Errorf("cpa client is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("auth file name is required")
	}
	if len(content) == 0 {
		return fmt.Errorf("auth file content is empty")
	}
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	switch c.uploadMode {
	case UploadModeJSON:
		return c.uploadAuthFileByJSON(ctx, name, content)
	case UploadModeMultipart:
		return c.uploadAuthFileByMultipart(ctx, name, content)
	default:
		return fmt.Errorf("unsupported upload mode: %s", c.uploadMode)
	}
}

func (c *Client) ListAuthFiles(ctx context.Context) (AuthFilesResponse, error) {
	var result AuthFilesResponse
	if c == nil {
		return result, fmt.Errorf("cpa client is nil")
	}
	if err := c.validate(); err != nil {
		return result, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint := c.baseURL + "/v0/management/auth-files"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Accept", "application/json")

	respBody, err := c.do(req)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return result, fmt.Errorf("unmarshal auth files response failed: %w", err)
	}

	return result, nil
}

func (c *Client) DownloadAuthFile(ctx context.Context, name string) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("cpa client is nil")
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("auth file name is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint := c.baseURL + "/v0/management/auth-files/download?name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Accept", "application/json")
	return c.do(req)
}

func (c *Client) DeleteAuthFile(ctx context.Context, name string) error {
	if c == nil {
		return fmt.Errorf("cpa client is nil")
	}
	if err := c.validate(); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("auth file name is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint := c.baseURL + "/v0/management/auth-files?name=" + url.QueryEscape(name)
	//endpoint := c.baseURL + "/auth-files?name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	_, err = c.do(req)
	return err
}

func (c *Client) SetAuthFileStatus(ctx context.Context, reqBody AuthFileStatusRequest) error {
	if c == nil {
		return fmt.Errorf("cpa client is nil")
	}
	if err := c.validate(); err != nil {
		return err
	}
	reqBody.Name = strings.TrimSpace(reqBody.Name)
	if reqBody.Name == "" {
		return fmt.Errorf("auth file name is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal auth file status failed: %w", err)
	}

	endpoint := c.baseURL + "/v0/management/auth-files/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Content-Type", "application/json")
	_, err = c.do(req)
	return err
}

func (c *Client) Usage(ctx context.Context) (UsageResponse, error) {
	var result UsageResponse
	if c == nil {
		return result, fmt.Errorf("cpa client is nil")
	}
	if err := c.validate(); err != nil {
		return result, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint := c.baseURL + "/usage"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Accept", "application/json")

	respBody, err := c.do(req)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return result, fmt.Errorf("unmarshal usage response failed: %w", err)
	}
	return result, nil
}

func (c *Client) uploadAuthFileByJSON(ctx context.Context, name string, content []byte) error {
	endpoint := c.baseURL + "/v0/management/auth-files?name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Content-Type", "application/json")
	_, err = c.do(req)
	return err
}

func (c *Client) uploadAuthFileByMultipart(ctx context.Context, name string, content []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", name)
	if err != nil {
		return err
	}
	if _, err := fileWriter.Write(content); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	endpoint := c.baseURL + "/v0/management/auth-files"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.managementKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, err = c.do(req)
	return err
}

func (c *Client) validate() error {
	if c.baseURL == "" {
		return fmt.Errorf("baseURL is required")
	}
	if c.managementKey == "" {
		return fmt.Errorf("managementKey is required")
	}
	return nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, buildResponseError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	return body, nil
}

func buildResponseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	err := errors.New(fmt.Sprintf("http status %d: %s", resp.StatusCode, message))
	return err
}
