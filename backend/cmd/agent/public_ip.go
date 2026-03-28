package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bigops/platform/internal/agent"
)

const defaultPublicIPProvider = "https://cip.cc"

type publicIPConfig struct {
	ConfiguredPublicIP string
	CacheFile          string
	ProviderURL        string
	Timeout            time.Duration
	RefreshInterval    time.Duration
	CheckInterval      time.Duration
	Now                func() time.Time
}

type publicIPCache struct {
	IP        string    `json:"ip"`
	FetchedAt time.Time `json:"fetched_at"`
}

func resolvePublicIPCacheFile(configPath, configuredCacheFile string) string {
	if strings.TrimSpace(configuredCacheFile) != "" {
		return configuredCacheFile
	}
	configDir := filepath.Dir(configPath)
	if configDir == "." || configDir == "" {
		configDir = "config"
	}
	return filepath.Join(configDir, ".agent-public-ip.json")
}

func resolvePublicIP(cfg publicIPConfig) (string, error) {
	if strings.TrimSpace(cfg.ConfiguredPublicIP) != "" {
		return strings.TrimSpace(cfg.ConfiguredPublicIP), nil
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	if cfg.ProviderURL == "" {
		cfg.ProviderURL = defaultPublicIPProvider
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 24 * time.Hour
	}

	cache, _ := loadPublicIPCache(cfg.CacheFile)
	if cache.IP != "" && nowFn().Sub(cache.FetchedAt) < cfg.RefreshInterval {
		return cache.IP, nil
	}

	ip, err := fetchPublicIP(cfg.ProviderURL, cfg.Timeout)
	if err != nil {
		if cache.IP != "" {
			return cache.IP, nil
		}
		return "", err
	}

	cache = publicIPCache{
		IP:        ip,
		FetchedAt: nowFn(),
	}
	if err := savePublicIPCache(cfg.CacheFile, cache); err != nil {
		return ip, nil
	}
	return ip, nil
}

func watchPublicIP(ctx context.Context, cfg publicIPConfig, client *agent.AgentClient) {
	if strings.TrimSpace(cfg.ConfiguredPublicIP) != "" || client == nil {
		return
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = time.Hour
	}
	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ip, err := resolvePublicIP(cfg)
			if err != nil {
				continue
			}
			client.SetPublicIP(ip)
		}
	}
}

func loadPublicIPCache(path string) (publicIPCache, error) {
	if strings.TrimSpace(path) == "" {
		return publicIPCache{}, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return publicIPCache{}, err
	}
	var cache publicIPCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return publicIPCache{}, err
	}
	return cache, nil
}

func savePublicIPCache(path string, cache publicIPCache) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func fetchPublicIP(providerURL string, timeout time.Duration) (string, error) {
	providerURL = strings.TrimSpace(providerURL)
	if providerURL == "" {
		return "", errors.New("no public ip provider configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, providerURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return "", readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("public ip provider returned non-2xx status")
	}

	if ip := parsePublicIPBody(body); ip != "" {
		return ip, nil
	}
	return "", errors.New("public ip provider returned invalid body")
}

func isValidIP(value string) bool {
	return net.ParseIP(strings.TrimSpace(value)) != nil
}

func parsePublicIPBody(body []byte) string {
	var jsonResp struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &jsonResp); err == nil && isValidIP(jsonResp.IP) {
		return strings.TrimSpace(jsonResp.IP)
	}

	raw := strings.TrimSpace(string(body))
	if isValidIP(raw) {
		return raw
	}

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if idx := strings.Index(trimmed, ":"); idx >= 0 {
			candidate := strings.TrimSpace(trimmed[idx+1:])
			if isValidIP(candidate) {
				return candidate
			}
		}
		for _, field := range strings.Fields(trimmed) {
			if isValidIP(field) {
				return field
			}
		}
	}
	return ""
}
