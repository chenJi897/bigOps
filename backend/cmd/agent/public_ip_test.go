package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolvePublicIP_UsesConfiguredValue(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "public-ip.json")
	ip, err := resolvePublicIP(publicIPConfig{
		ConfiguredPublicIP: "1.2.3.4",
		CacheFile:          cacheFile,
		ProviderURL:        "http://127.0.0.1:1",
		Timeout:            time.Second,
		RefreshInterval:    24 * time.Hour,
		Now:                func() time.Time { return time.Unix(1000, 0) },
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ip != "1.2.3.4" {
		t.Fatalf("expected configured public ip, got %q", ip)
	}
}

func TestResolvePublicIP_FetchesAndPersistsOnFirstRun(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"ip":"8.8.8.8"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "public-ip.json")
	ip, err := resolvePublicIP(publicIPConfig{
		CacheFile:       cacheFile,
		ProviderURL:     server.URL,
		Timeout:         time.Second,
		RefreshInterval: 24 * time.Hour,
		Now:             func() time.Time { return time.Unix(2000, 0) },
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ip != "8.8.8.8" {
		t.Fatalf("expected fetched public ip, got %q", ip)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected single provider call, got %d", calls.Load())
	}

	ip2, err := resolvePublicIP(publicIPConfig{
		CacheFile:       cacheFile,
		ProviderURL:     server.URL,
		Timeout:         time.Second,
		RefreshInterval: 24 * time.Hour,
		Now:             func() time.Time { return time.Unix(2000 + 3600, 0) },
	})
	if err != nil {
		t.Fatalf("expected no error on cache reuse, got %v", err)
	}
	if ip2 != "8.8.8.8" {
		t.Fatalf("expected cached public ip, got %q", ip2)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected cache hit without extra provider call, got %d", calls.Load())
	}
}

func TestResolvePublicIP_RefreshesAfterTTL(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := calls.Add(1)
		if current == 1 {
			_, _ = w.Write([]byte(`{"ip":"8.8.4.4"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ip":"1.1.1.1"}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "public-ip.json")
	first, err := resolvePublicIP(publicIPConfig{
		CacheFile:       cacheFile,
		ProviderURL:     server.URL,
		Timeout:         time.Second,
		RefreshInterval: 24 * time.Hour,
		Now:             func() time.Time { return time.Unix(3000, 0) },
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if first != "8.8.4.4" {
		t.Fatalf("expected first ip, got %q", first)
	}

	second, err := resolvePublicIP(publicIPConfig{
		CacheFile:       cacheFile,
		ProviderURL:     server.URL,
		Timeout:         time.Second,
		RefreshInterval: 24 * time.Hour,
		Now:             func() time.Time { return time.Unix(3000 + int64((25 * time.Hour).Seconds()), 0) },
	})
	if err != nil {
		t.Fatalf("expected no error after ttl refresh, got %v", err)
	}
	if second != "1.1.1.1" {
		t.Fatalf("expected refreshed public ip, got %q", second)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected second provider call after ttl, got %d", calls.Load())
	}
}

func TestResolvePublicIP_ParsesCipCCBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("IP\t: 222.128.28.198\n地址\t: 中国 北京 北京\n运营商\t: 联通\n"))
	}))
	defer server.Close()

	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "public-ip.json")
	ip, err := resolvePublicIP(publicIPConfig{
		CacheFile:       cacheFile,
		ProviderURL:     server.URL,
		Timeout:         time.Second,
		RefreshInterval: 24 * time.Hour,
		Now:             func() time.Time { return time.Unix(4000, 0) },
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ip != "222.128.28.198" {
		t.Fatalf("expected cip.cc ip, got %q", ip)
	}
}
