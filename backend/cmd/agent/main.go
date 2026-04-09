package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bigops/platform/internal/agent"
	"github.com/spf13/viper"
)

func main() {
	configPath := flag.String("config", "config/agent.yaml", "config file path")
	flag.Parse()

	viper.SetConfigFile(*configPath)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	serverAddr := viper.GetString("server.address")
	hostname := viper.GetString("agent.hostname")
	configuredAgentID := viper.GetString("agent.id")
	configuredStateFile := viper.GetString("agent.state_file")
	publicIP := viper.GetString("agent.public_ip")
	publicIPProvider := viper.GetString("agent.public_ip_provider")
	publicIPCacheFile := viper.GetString("agent.public_ip_cache_file")
	publicIPTimeoutSeconds := viper.GetInt("agent.public_ip_timeout_seconds")
	publicIPRefreshHours := viper.GetInt("agent.public_ip_refresh_hours")

	// Resource limits (inspired by didi/falcon-log-agent)
	maxCPURate := viper.GetFloat64("agent.max_cpu_rate")
	maxMemMB := viper.GetInt("agent.max_mem_mb")
	if maxCPURate > 0 {
		cores := agent.ApplyCPULimit(maxCPURate)
		log.Printf("CPU limit applied: GOMAXPROCS=%d (rate=%.0f%%)", cores, maxCPURate*100)
	}
	if maxMemMB > 0 {
		agent.StartMemoryPatrol(maxMemMB, 10*time.Second)
		log.Printf("Memory patrol started: limit=%dMB", maxMemMB)
	}
	if hostname == "" {
		h, _ := os.Hostname()
		hostname = h
	}

	privateIP := getLocalIP()
	stateFile := resolveAgentStateFile(*configPath, configuredStateFile)
	publicIPCachePath := resolvePublicIPCacheFile(*configPath, publicIPCacheFile)
	agentID, err := resolveAgentID(configuredAgentID, stateFile)
	if err != nil {
		log.Fatalf("Failed to resolve agent id: %v", err)
	}
	publicIPCfg := publicIPConfig{
		ConfiguredPublicIP: publicIP,
		CacheFile:          publicIPCachePath,
		ProviderURL:        publicIPProvider,
		Timeout:            time.Duration(publicIPTimeoutSeconds) * time.Second,
		RefreshInterval:    time.Duration(publicIPRefreshHours) * time.Hour,
		CheckInterval:      time.Hour,
	}
	resolvedPublicIP, err := resolvePublicIP(publicIPCfg)
	if err != nil {
		log.Printf("Resolve public ip failed: %v", err)
	}
	publicIP = resolvedPublicIP

	log.Printf("BigOps Agent starting, server=%s hostname=%s private_ip=%s public_ip=%s agent_id=%s", serverAddr, hostname, privateIP, publicIP, agentID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := agent.NewAgentClient(serverAddr, agentID, hostname, privateIP, publicIP)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.Close()

	go client.Run(ctx)
	go watchPublicIP(ctx, publicIPCfg, client)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Agent shutting down...")
	cancel()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}
