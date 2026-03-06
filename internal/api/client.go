package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"umbraco-cli/internal/auth"
	"umbraco-cli/internal/config"
	"umbraco-cli/internal/validate"
)

type RequestOptions struct {
	Fields string
	Params map[string]any
	DryRun bool
}

type DryRunResult struct {
	DryRun bool   `json:"dryRun"`
	Valid  bool   `json:"valid"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   any    `json:"body"`
}

type Client struct {
	cfg           config.Config
	httpClient    *http.Client
	tokenProvider *auth.Provider
}

func NewClient(cfg config.Config, httpClient *http.Client, tokenProvider *auth.Provider) *Client {
	return &Client{cfg: cfg, httpClient: httpClient, tokenProvider: tokenProvider}
}

func (c *Client) buildURL(path string, opts RequestOptions) (string, error) {
	normalizedPath := path
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/umbraco/management/api/v1" + normalizedPath

	query := base.Query()
	if opts.Fields != "" {
		if err := validate.String(opts.Fields); err != nil {
			return "", err
		}
		query.Set("fields", opts.Fields)
	}
	if opts.Params != nil {
		if err := validate.Input(opts.Params); err != nil {
			return "", err
		}
		for key, raw := range opts.Params {
			if raw == nil {
				continue
			}
			switch value := raw.(type) {
			case []any:
				for _, item := range value {
					query.Add(key, fmt.Sprint(item))
				}
			default:
				query.Add(key, fmt.Sprint(value))
			}
		}
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func parseResponse(resp *http.Response) (any, error) {
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}

	if strings.Contains(contentType, "application/json") {
		var payload any
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err == nil {
		return payload, nil
	}
	return string(body), nil
}

func (c *Client) Request(ctx context.Context, method string, path string, body any, opts RequestOptions) (any, error) {
	if raw, ok := body.(map[string]any); ok {
		if err := validate.Input(raw); err != nil {
			return nil, err
		}
	}

	fullURL, err := c.buildURL(path, opts)
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		relative := strings.TrimPrefix(fullURL, strings.TrimRight(c.cfg.BaseURL, "/"))
		return DryRunResult{
			DryRun: true,
			Valid:  true,
			Method: method,
			Path:   relative,
			Body:   body,
		}, nil
	}

	token, err := c.tokenProvider.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		encoded, _ := json.Marshal(result)
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, encoded)
	}

	return result, nil
}

func (c *Client) Get(ctx context.Context, path string, opts RequestOptions) (any, error) {
	return c.Request(ctx, http.MethodGet, path, nil, opts)
}

func (c *Client) Post(ctx context.Context, path string, body any, opts RequestOptions) (any, error) {
	return c.Request(ctx, http.MethodPost, path, body, opts)
}

func (c *Client) Put(ctx context.Context, path string, body any, opts RequestOptions) (any, error) {
	return c.Request(ctx, http.MethodPut, path, body, opts)
}

func (c *Client) Delete(ctx context.Context, path string, opts RequestOptions) (any, error) {
	return c.Request(ctx, http.MethodDelete, path, nil, opts)
}
