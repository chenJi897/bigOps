package service

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

type PrometheusClient struct {
	baseURL  string
	username string
	password string
	headers  map[string]string
	client   *http.Client
}

func NewPrometheusClient(baseURL, username, password string, headers map[string]string) *PrometheusClient {
	return &PrometheusClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		headers:  headers,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *PrometheusClient) HealthCheck(ctx context.Context) error {
	_, err := c.doRequest(ctx, "/api/v1/targets", nil)
	return err
}

func (c *PrometheusClient) Query(ctx context.Context, query string, ts time.Time) (map[string]any, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("time", fmt.Sprintf("%s", ts.Format(time.RFC3339Nano)))
	return c.doRequest(ctx, "/api/v1/query", params)
}

func (c *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (map[string]any, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start.Format(time.RFC3339Nano))
	params.Set("end", end.Format(time.RFC3339Nano))
	params.Set("step", fmt.Sprintf("%.0f", step.Seconds()))
	return c.doRequest(ctx, "/api/v1/query_range", params)
}

func (c *PrometheusClient) doRequest(ctx context.Context, path string, params url.Values) (map[string]any, error) {
	if params == nil {
		params = url.Values{}
	}
	endpoint := c.baseURL + path
	if encoded := params.Encode(); encoded != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus response %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}
